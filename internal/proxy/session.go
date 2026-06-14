package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"proxy-core/internal/db"
	"proxy-core/internal/models"
)

func forceCleanJobs(jobJSON string) string {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(jobJSON), &msg); err == nil {
		if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
			params[8] = true
			if modBytes, err := json.Marshal(msg); err == nil {
				return string(modBytes)
			}
		}
	}
	return jobJSON
}

type FeeMode string

const (
	FeeModeNone     FeeMode = "NONE"
	FeeModeDev      FeeMode = "DEV"
	FeeModeOperator FeeMode = "OPERATOR"
)

type SessionStats struct {
	Shares        int64
	FeeShares     int64
	ValidShares   int64
	InvalidShares int64
	ConnectedAt   time.Time
}

type ShareEvent struct {
	Timestamp time.Time
	Diff      float64
}

// PendingTracker (LRU) to prevent memory leak from unreplied shares
type PendingTracker struct {
	mu     sync.Mutex
	shares map[interface{}]string
	order  []interface{}
}

func NewPendingTracker() *PendingTracker {
	return &PendingTracker{
		shares: make(map[interface{}]string),
		order:  make([]interface{}, 0),
	}
}

func (t *PendingTracker) Store(id interface{}, reqLine string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.shares[id]; !exists {
		t.order = append(t.order, id)
		if len(t.order) > 1000 {
			oldest := t.order[0]
			t.order = t.order[1:]
			delete(t.shares, oldest)
		}
	}
	t.shares[id] = reqLine
}

func (t *PendingTracker) LoadAndDelete(id interface{}) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if val, exists := t.shares[id]; exists {
		delete(t.shares, id)
		return val, true
	}
	return "", false
}

func (t *PendingTracker) Delete(id interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.shares, id)
}

type Session struct {
	ID        string
	MinerConn net.Conn
	MainConn  net.Conn
	FeeConn   net.Conn
	Config    *models.ProxyConfig
	Server    *Server

	MinerWallet string
	MinerWorker string
	ClientAgent string // Firmware or Miner Software version
	Protocol    string // "STRATUM" or "ETH_PROXY"

	State          string // "MAIN", "SWITCHING_TO_FEE", "FEE", "SWITCHING_TO_MAIN"
	TargetState    string
	CurrentFeeMode FeeMode

	Stats SessionStats

	ShareHistory   []ShareEvent
	CurrentDiff    float64
	RemoteDiff     float64
	LocalDiff      float64
	LastHashUpdate time.Time
	LastShareTime  time.Time
	DisplayHash    float64
	PeakHash       float64
	IsEncrypted    bool

	IsOffline      bool
	OfflineAt      time.Time

	LastExtranonceCmdTime time.Time
	IsBuggyAsic           bool

	// Locks and sync
	mu   sync.Mutex
	quit chan struct{}

	// Protocols
	loginPackets  []map[string]interface{}
	pendingShares *PendingTracker
	cycleOffset   int

	// Smart Fee Routing & Fallback
	FeeAuthFailures   int
	SamePoolFeeActive bool

	// JobTracker (LRU) to prevent memory leak
	jobTracker map[string]bool
	jobList    []string

	// ASIC Extranonce Support
	SubscribeID    interface{}
	MainExtranonce *ExtranonceData
	FeeExtranonce  *ExtranonceData

	// Zero-Latency Switching
	LatestMainJob string
	LatestFeeJob  string
	IsPreWarmed   bool

	// ASIC Optimizations State
	currentMainJob        string
	currentFeeJob         string
	currentMainTargetHash string
}

func NewSession(conn net.Conn, cfg *models.ProxyConfig, isEncrypted bool) *Session {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return &Session{
		ID:             id,
		MinerConn:      conn,
		Config:         cfg,
		State:          "MAIN",
		TargetState:    "MAIN",
		CurrentFeeMode: FeeModeNone,
		Stats:          SessionStats{ConnectedAt: time.Now()},
		quit:           make(chan struct{}),
		loginPackets:   make([]map[string]interface{}, 0),
		pendingShares:  NewPendingTracker(),
		cycleOffset:    -1,
		ShareHistory:   make([]ShareEvent, 0),
		CurrentDiff:    1.0,
		LastHashUpdate: time.Now(),
		LastShareTime:  time.Now(),
		jobTracker:     make(map[string]bool),
		jobList:        make([]string, 0),
		IsEncrypted:    isEncrypted,
	}
}

func parseEthProxyTargetToDiff(targetHex string) float64 {
	targetHex = strings.TrimPrefix(targetHex, "0x")
	tInt, ok := new(big.Int).SetString(targetHex, 16)
	if !ok || tInt.Sign() == 0 {
		return 1.0
	}
	tFloat := new(big.Float).SetInt(tInt)
	maxT := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil))
	hashFloat := new(big.Float).Quo(maxT, tFloat)
	diffFloat := new(big.Float).Quo(hashFloat, big.NewFloat(4294967296.0))
	diff, _ := diffFloat.Float64()
	if diff <= 0 {
		return 1.0
	}
	return diff
}

const MaxTrackedJobs = 10000

func (s *Session) addJob(jobID string, isMain bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobTracker[jobID]; !exists {
		s.jobList = append(s.jobList, jobID)
		if len(s.jobList) > MaxTrackedJobs {
			oldest := s.jobList[0]
			s.jobList = s.jobList[1:]
			delete(s.jobTracker, oldest)
		}
	}
	s.jobTracker[jobID] = isMain
	if isMain {
		s.currentMainJob = jobID
	} else {
		s.currentFeeJob = jobID
	}
}

func (s *Session) checkJobIsMain(jobID string) (bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	isMain, exists := s.jobTracker[jobID]
	return isMain, exists
}

func (s *Session) getWorkerKey() string {
	if s.MinerWorker != "" {
		return s.MinerWorker
	}
	return s.MinerConn.RemoteAddr().String()
}

func (s *Session) LogGeneral(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if s.Server != nil {
		s.Server.GetLogger(s.getWorkerKey()).AddLog(LogTypeGeneral, msg, s.Config.EnableDetailedLog)
	}
}

func (s *Session) LogError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if s.Server != nil {
		s.Server.GetLogger(s.getWorkerKey()).AddLog(LogTypeError, msg, s.Config.EnableDetailedLog)
	}
}

func (s *Session) Start() {
	encTag := ""
	if s.IsEncrypted {
		encTag = "[隧道加密🛡️] "
	}
	s.LogGeneral("%sConnected from %s", encTag, s.MinerConn.RemoteAddr().String())

	// Connect to main pool
	var err error
	s.MainConn, err = net.Dial("tcp", s.Config.PoolAddress)
	if err != nil {
		s.LogError("Failed to connect to main pool: %v", err)
		s.Close()
		return
	}
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(s.MainConn)
	}

	go s.timerLoop()
	go s.StartVardiffEngine()
	go s.Watchdog()
	go s.readMainLoop()
	s.readMinerLoop()
}

func (s *Session) Close() {
	s.LogGeneral("Session Close called")
	s.mu.Lock()
	
	lastExt := s.LastExtranonceCmdTime
	isBuggy := s.IsBuggyAsic
	enableAuto := s.Config.EnableAutoQuarantine
	
	ident := ""
	if s.MinerWallet != "" && s.MinerWorker != "" {
		ident = fmt.Sprintf("%s.%s", s.MinerWallet, s.MinerWorker)
	} else if s.MinerWorker != "" {
		ident = s.MinerWorker
	} else if s.MinerWallet != "" {
		ident = s.MinerWallet
	} else if s.MinerConn != nil {
		ident = s.MinerConn.RemoteAddr().String()
	}

	defer s.mu.Unlock()

	select {
	case <-s.quit:
		return
	default:
		close(s.quit)
	}

	if enableAuto && !isBuggy && !lastExt.IsZero() && time.Since(lastExt) < 15*time.Second {
		s.IsBuggyAsic = true
		s.LogGeneral("🤖 [AI-Quarantine] ASIC TCP Drop Detected (Disconnected within 15s of command). Auto-Quarantining: %s", ident)
		_ = db.AddSafeMiner(s.Config.ListenPort, ident)
		
		if s.Config.SafeMiners == "" {
			s.Config.SafeMiners = ident
		} else {
			s.Config.SafeMiners += "," + ident
		}
	}

	if s.MinerConn != nil {
		s.MinerConn.Close()
	}
	if s.MainConn != nil {
		s.MainConn.Close()
	}
	if s.FeeConn != nil {
		s.FeeConn.Close()
	}
}

// getAlgoBaseMHs automatically detects the coin algorithm from the pool address
// and returns the appropriate difficulty-to-MH/s multiplier.
func (s *Session) getAlgoBaseMHs() float64 {
	// Default SHA-256 (BTC, BCH) base multiplier (GH/s base)
	base := 4.294967296

	pool := strings.ToLower(s.Config.PoolAddress)
	coin := strings.ToLower(s.Config.CoinName)

	if strings.Contains(pool, "ltc") || strings.Contains(pool, "doge") || strings.Contains(coin, "ltc") {
		// Scrypt (LTC/DOGE) uses 2^16 instead of 2^32. Ratio is 1 / 65536
		base = 4.294967296 / 65536.0
	} else if strings.Contains(pool, "prl") || strings.Contains(coin, "prl") {
		// Pearlhash (PRL) multiplier calibration
		// k1pool uses a custom high-difficulty base where Diff 1.0 = 2.95 PH
		if strings.Contains(pool, "k1pool") {
			base = 4.294967296 * 688256.0
		} else {
			// alphapool and others use the standard difficulty base (Diff 1 = 4.29 GH)
			base = 4.294967296
		}
	} else if strings.Contains(pool, "etc") || strings.Contains(coin, "etc") || strings.Contains(pool, "eth") {
		// Ethash/Etchash (ETC/ETHW) uses the standard 2^32 hashrate scale (1 diff = 4.29 GH)
		base = 4.294967296
	}

	return base * 1000 // Convert to MH/s base
}

func (s *Session) FormatHashrate() string {
	// If HashrateMultiplier is set, fake it
	multiplier := s.Config.HashrateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// Calculate rolling window hashrate (10 minutes)
	now := time.Now()
	// Reverted to 10 minutes to save CPU under massive concurrency (10,000+ miners)
	updateInterval := 10 * time.Minute
	uptimeSecs := now.Sub(s.Stats.ConnectedAt).Seconds()

	if uptimeSecs < 600 {
		updateInterval = 1 * time.Minute
	}

	if now.Sub(s.LastHashUpdate) >= updateInterval || s.DisplayHash == 0 {
		s.mu.Lock()
		// Filter last 10 minutes
		cutoff := now.Add(-10 * time.Minute)
		filtered := make([]ShareEvent, 0)
		var diffSum float64 = 0

		for _, ev := range s.ShareHistory {
			if ev.Timestamp.After(cutoff) {
				filtered = append(filtered, ev)
				diffSum += ev.Diff
			}
		}
		s.ShareHistory = filtered

		window := uptimeSecs
		if window > 600 {
			window = 600
		}
		if window <= 0 {
			window = 1
		}

		// Auto-detect algorithm base multiplier based on pool address
		algoBase := s.getAlgoBaseMHs()
		s.DisplayHash = (diffSum * algoBase) / window
		if s.DisplayHash > s.PeakHash {
			s.PeakHash = s.DisplayHash
		}
		s.LastHashUpdate = now
		s.mu.Unlock()
	}

	hs := s.DisplayHash * multiplier

	unit := s.Config.HashrateUnit
	if unit == "" {
		if hs >= 1000000000 {
			hs = hs / 1000000000
			unit = "PH/s"
		} else if hs >= 1000000 {
			hs = hs / 1000000
			unit = "TH/s"
		} else if hs >= 1000 {
			hs = hs / 1000
			unit = "GH/s"
		} else {
			unit = "MH/s"
		}
	}

	return fmt.Sprintf("%.2f %s", hs, unit)
}

// readMinerLoop reads lines from the miner
func (s *Session) readMinerLoop() {
	defer s.Close()
	scanner := bufio.NewScanner(s.MinerConn)
	bufPtr := ScannerBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	defer ScannerBufferPool.Put(bufPtr)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if s.Config.EnableDetailedLog {
			s.LogGeneral("[RAW MINER RX] %s", strings.TrimSpace(line))
		}

		var method string
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			method, _ = msg["method"].(string)
			if method == "mining.subscribe" || method == "eth_submitLogin" || method == "mining.authorize" || method == "login" || method == "mining.configure" || method == "eth_submitHashrate" {
				// Deep copy msg to store in loginPackets so it isn't mutated by MainFixedDifficulty
				var pktCopy map[string]interface{}
				pktBytes, _ := json.Marshal(msg)
				_ = json.Unmarshal(pktBytes, &pktCopy)
				s.loginPackets = append(s.loginPackets, pktCopy)
				if method == "mining.subscribe" {
					s.mu.Lock()
					s.SubscribeID = msg["id"]
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if agent, ok := params[0].(string); ok {
							s.ClientAgent = agent
							if s.MinerConn != nil {
								if tcpAddr, ok := s.MinerConn.RemoteAddr().(*net.TCPAddr); ok {
									s.Server.ClientAgentCache.Store(tcpAddr.IP.String(), agent)
								}
							}
						}
					}
					s.mu.Unlock()
				}
				if method == "mining.authorize" || method == "eth_submitLogin" || method == "login" {
					// Check for root-level "client" field
					if clientStr, ok := msg["client"].(string); ok && clientStr != "" {
						s.ClientAgent = clientStr
					}
					if method == "eth_submitLogin" {
						s.Protocol = "ETH_PROXY"
						if s.ClientAgent == "" && s.MinerConn != nil {
							if tcpAddr, ok := s.MinerConn.RemoteAddr().(*net.TCPAddr); ok {
								if cachedAgent, exists := s.Server.ClientAgentCache.Load(tcpAddr.IP.String()); exists {
									s.ClientAgent = cachedAgent.(string)
								}
							}
						}
					} else {
						s.Protocol = "STRATUM"
					}

					// --- SMART DPI COIN VALIDATION ---
					expectedCoin := strings.ToUpper(s.Config.CoinName)
					isEthFamily := (expectedCoin == "ETC" || expectedCoin == "ETHW" || expectedCoin == "PRL")
					if isEthFamily && s.Protocol != "ETH_PROXY" {
						s.LogGeneral("[Anti-Cheat] Miner sent STRATUM protocol but port configured for %s. Dropping connection.", expectedCoin)
						s.Close()
						return
					} else if !isEthFamily && s.Protocol != "STRATUM" {
						s.LogGeneral("[Anti-Cheat] Miner sent ETH_PROXY protocol but port configured for %s. Dropping connection.", expectedCoin)
						s.Close()
						return
					}
					// ---------------------------------
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						// 1. Check for root-level "worker" field (standard for many ASICs/ETH-Proxy)
						if workerRoot, hasWorker := msg["worker"].(string); hasWorker && workerRoot != "" {
							s.MinerWorker = workerRoot
						}

						if pStr, ok := params[0].(string); ok {
							parts := strings.Split(pStr, ".")
							s.MinerWallet = parts[0]
							
							// 2. If worker wasn't found at the root level, try to extract from params
							if s.MinerWorker == "" {
								if len(parts) > 1 {
									s.MinerWorker = parts[1]
								} else if len(params) > 1 {
									if p1Str, ok := params[1].(string); ok && p1Str != "x" && p1Str != "" && p1Str != "password" && !strings.HasPrefix(p1Str, "d=") {
										s.MinerWorker = p1Str
									} else {
										s.MinerWorker = "worker"
									}
								} else {
									s.MinerWorker = "worker"
								}
							}
						}
					} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
						if w, ok := paramsMap["wallet"].(string); ok {
							s.MinerWallet = w
						}
						if w, ok := paramsMap["worker"].(string); ok {
							s.MinerWorker = w
						}
					}

					// Sanitize miner worker to avoid upstream rejection
					if s.MinerWorker == "" || s.MinerWorker == "(null)" || s.MinerWorker == "null" {
						s.MinerWorker = "default"
					}
					s.MinerWorker = strings.ReplaceAll(s.MinerWorker, "(", "")
					s.MinerWorker = strings.ReplaceAll(s.MinerWorker, ")", "")

					// Rewrite params to ensure the upstream pool receives the sanitized worker name
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if _, ok := params[0].(string); ok {
							if s.MinerWallet != "" {
								params[0] = fmt.Sprintf("%s.%s", s.MinerWallet, s.MinerWorker)
							} else {
								params[0] = s.MinerWorker
							}
							if modBytes, err := json.Marshal(msg); err == nil {
								line = string(modBytes)
							}
						}
					}

					if s.Server != nil && s.MinerWorker != "" {
						currentIP := ""
						if s.MinerConn != nil {
							if tcpAddr, ok := s.MinerConn.RemoteAddr().(*net.TCPAddr); ok {
								currentIP = tcpAddr.IP.String()
							}
						}
						oldSession := s.Server.CleanOfflineWorker(s.MinerWorker, currentIP)
						if oldSession != nil {
							oldSession.mu.Lock()
							oldStats := oldSession.Stats
							oldShareHistory := oldSession.ShareHistory
							oldLastShareTime := oldSession.LastShareTime
							oldSession.mu.Unlock()

							s.mu.Lock()
							s.Stats.Shares = oldStats.Shares
							s.Stats.ValidShares = oldStats.ValidShares
							s.Stats.InvalidShares = oldStats.InvalidShares
							s.Stats.FeeShares = oldStats.FeeShares
							s.Stats.ConnectedAt = oldStats.ConnectedAt
							s.ShareHistory = append([]ShareEvent{}, oldShareHistory...)
							if !oldLastShareTime.IsZero() {
								s.LastShareTime = oldLastShareTime
							}
							s.mu.Unlock()
							s.LogGeneral("Miner session restored from offline state, inherited %d valid shares", oldStats.ValidShares)
						} else {
							s.LogGeneral("Miner authorized: %s", s.MinerWorker)
						}
					}

					// Inject fixed difficulty
					// Inject main fixed difficulty
					mainDiff := s.Config.MainFixedDifficulty
					if mainDiff == "auto" {
						s.mu.Lock()
						curDiff := s.CurrentDiff
						s.mu.Unlock()
						if curDiff > 0 {
							mainDiff = fmt.Sprintf("d=%.0f", curDiff)
						} else {
							mainDiff = ""
						}
					}

					if mainDiff != "" {
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if len(params) > 1 {
								params[1] = mainDiff
							} else {
								msg["params"] = append(params, mainDiff)
							}
						} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
							paramsMap["pass"] = mainDiff
							paramsMap["password"] = mainDiff
						}
						// re-serialize line so mainConn gets the spoofed password
						if modBytes, err := json.Marshal(msg); err == nil {
							line = string(modBytes)
						}
					}
				}
			} else if method == "mining.submit" || method == "eth_submitWork" {
				s.Stats.Shares++
				s.mu.Lock()
				s.LastShareTime = time.Now()
				s.mu.Unlock()
				if id, ok := msg["id"]; ok {
					s.pendingShares.Store(id, strings.TrimSpace(line))
				}
			}
		}

		s.mu.Lock()
		state := s.State
		feeConn := s.FeeConn
		mainConn := s.MainConn
		s.mu.Unlock()

		if method == "mining.submit" || method == "eth_submitWork" {
			// Extract Job ID
			var submitJobID string
			if method == "mining.submit" {
				if params, ok := msg["params"].([]interface{}); ok && len(params) > 1 {
					if jobIDStr, ok := params[1].(string); ok {
						submitJobID = jobIDStr
					}
				}
			} else {
				if params, ok := msg["params"].([]interface{}); ok && len(params) > 1 {
					if jobIDStr, ok := params[1].(string); ok {
						submitJobID = jobIDStr
					}
				}
			}

			isMainRoute := (state == "MAIN" || state == "SWITCHING_TO_MAIN")
			if submitJobID != "" {
				if isMainRaw, exists := s.checkJobIsMain(submitJobID); exists {
					isMainRoute = isMainRaw
				}
			}

			if isMainRoute {
				if mainConn != nil {
					// Vardiff Fake Accept logic check
					s.mu.Lock()
					enableVardiff := s.Config.EnableVardiff
					localDiff := s.LocalDiff
					remoteDiff := s.RemoteDiff
					s.mu.Unlock()

					// If Vardiff is enabled and LocalDiff is less than RemoteDiff, we must evaluate fake accepts.
					// Since we don't have a full block hash calculator built-in yet, we use a probabilistic fake-accept
					// based on the ratio, OR simply if we forced difficulty UP (Local >= Remote), all shares are valid.
					shouldFakeAccept := false
					if enableVardiff {
						if localDiff > 0 && localDiff < remoteDiff && remoteDiff > 0 {
							// Probabilistic filter: only forward (LocalDiff / RemoteDiff) fraction of shares
							if rand.Float64() > (localDiff / remoteDiff) {
								shouldFakeAccept = true
							}
						}
					}

					// Stale Share Drop Optimization
					if s.Config.EnableStaleDrop {
						s.mu.Lock()
						activeMain := s.currentMainJob
						s.mu.Unlock()
						// Check if it's an ETH_PROXY style hash or standard Job ID
						if activeMain != "" && submitJobID != "" && submitJobID != activeMain {
							// For stratum, it's exact match. For eth_proxy, it's exact match on headerHash.
							shouldFakeAccept = true
						}
					}

					if shouldFakeAccept {
						if id, ok := msg["id"]; ok {
							s.pendingShares.Delete(id)
							fakeReply := fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
							_, _ = s.MinerConn.Write([]byte(fakeReply))
						}
					} else {
						fmt.Fprintf(mainConn, "%s\n", line)
					}
				}
			} else {
				if feeConn != nil {
					shouldFakeAccept := false
					// Stale Share Drop Optimization for Fee Pool
					if s.Config.EnableStaleDrop {
						s.mu.Lock()
						activeFee := s.currentFeeJob
						s.mu.Unlock()
						if activeFee != "" && submitJobID != "" && submitJobID != activeFee {
							shouldFakeAccept = true
						}
					}
					
					if shouldFakeAccept {
						if id, ok := msg["id"]; ok {
							s.pendingShares.Delete(id)
							fakeReply := fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
							_, _ = s.MinerConn.Write([]byte(fakeReply))
						}
					} else {
						fmt.Fprintf(feeConn, "%s\n", line)
					}
				} else {
					// Fee pool disconnected, rescue via fake accept
					if id, ok := msg["id"]; ok {
						s.pendingShares.Delete(id)
						fakeReply := fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
						_, _ = s.MinerConn.Write([]byte(fakeReply))
						s.LogGeneral("Perfect Routing: Fake accepted late fee share (%s) because fee connection is closed", submitJobID)
					}
				}
			}
		} else {
			// Non-submit packets route by current state
			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if feeConn != nil {
					fmt.Fprintf(feeConn, "%s\n", line)
				}
			} else {
				if mainConn != nil {
					fmt.Fprintf(mainConn, "%s\n", line)
				}
			}
		}
	}
}

func (s *Session) readMainLoop() {
	defer s.Close()
	scanner := bufio.NewScanner(s.MainConn)
	bufPtr := ScannerBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	defer ScannerBufferPool.Put(bufPtr)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		if s.Config.EnableDetailedLog {
			s.LogGeneral("[RAW MAIN RX] %s", strings.TrimSpace(line))
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			if s.Config.EnableAsic {
				s.mu.Lock()
				subId := s.SubscribeID
				s.mu.Unlock()
				if id, ok := msg["id"]; ok && id != nil && id == subId {
					if result, ok := msg["result"].([]interface{}); ok && len(result) > 2 {
						if en1, ok := result[1].(string); ok {
							if en2size, ok := result[2].(float64); ok {
								s.mu.Lock()
								s.MainExtranonce = &ExtranonceData{En1: en1, En2Size: int(en2size)}
								s.mu.Unlock()
							}
						}
					}
				}
				if method, ok := msg["method"].(string); ok && method == "mining.set_extranonce" {
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 1 {
						if en1, ok := params[0].(string); ok {
							if en2size, ok := params[1].(float64); ok {
								s.mu.Lock()
								s.MainExtranonce = &ExtranonceData{En1: en1, En2Size: int(en2size)}
								s.mu.Unlock()
							}
						}
					}
				}
			}

			if id, ok := msg["id"]; ok && id != nil {
				if origReq, isShareReply := s.pendingShares.LoadAndDelete(id); isShareReply {
					isReject := false
					if errObj, ok := msg["error"]; ok && errObj != nil {
						s.Stats.InvalidShares++
						isReject = true
					} else if res, ok := msg["result"]; ok && res == false {
						s.Stats.InvalidShares++
						isReject = true
					} else if res, ok := msg["result"]; ok && res == true {
						s.Stats.ValidShares++
						s.mu.Lock()
						s.ShareHistory = append(s.ShareHistory, ShareEvent{
							Timestamp: time.Now(),
							Diff:      s.CurrentDiff,
						})
						s.mu.Unlock()
					}

					if isReject {
						if s.Config.EnableDetailedLog {
							s.LogError("[MAIN] share rejected! Pool Response: %s | Original Request: %s", strings.TrimSpace(line), origReq)
						} else {
							s.LogError("[MAIN] share rejected! %s", strings.TrimSpace(line))
						}
						s.mu.Lock()
						antiBan := s.Config.EnableAntiBan
						s.mu.Unlock()
						if antiBan {
							line = fmt.Sprintf(`{"id": %v, "result": true, "error": null}`, id)
						}
					} else {
						s.LogGeneral("[MAIN] share accepted! [Diff: %.4f]", s.CurrentDiff)
					}
				} else {
					// Check for eth_getWork response
					if resArr, ok := msg["result"].([]interface{}); ok && len(resArr) >= 3 {
						if powHash, ok := resArr[0].(string); ok && strings.HasPrefix(powHash, "0x") {
							s.addJob(powHash, true) // true = Main
							if targetHash, ok := resArr[2].(string); ok && strings.HasPrefix(targetHash, "0x") {
								diffVal := parseEthProxyTargetToDiff(targetHash)
								s.mu.Lock()
								s.currentMainTargetHash = targetHash
								s.CurrentDiff = diffVal
								s.RemoteDiff = diffVal
								s.mu.Unlock()
							}
						}
					}
				}
			} else if method, ok := msg["method"].(string); ok {
				if method == "mining.set_difficulty" {
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if diffFloat, ok := params[0].(float64); ok {
							s.mu.Lock()
							s.CurrentDiff = diffFloat
							s.RemoteDiff = diffFloat
							enableVardiff := s.Config.EnableVardiff
							s.mu.Unlock()
							GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

							if enableVardiff && s.LocalDiff > 0 {
								// INTERCEPT: Do not forward to miner. Let VardiffEngine handle local difficulty.
								// Only intercept if we actually have a LocalDiff set, otherwise the miner mines blind.
								continue
							}
						}
					}
				} else if method == "mining.notify" {
					GlobalDispatcher.UpdateJob(s.Config.PoolAddress, line)
					s.mu.Lock()
					s.LatestMainJob = line
					s.mu.Unlock()
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if jobID, ok := params[0].(string); ok {
							s.addJob(jobID, true) // true = Main
						}
					}
				}
			}
		}

		s.mu.Lock()
		state := s.State
		minerConn := s.MinerConn
		s.mu.Unlock()

		if state == "MAIN" || state == "SWITCHING_TO_MAIN" {
			if minerConn != nil {
				fmt.Fprintf(minerConn, "%s\n", line)
			}
		}
	}
}

func (s *Session) timerLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.mu.Lock()
			devPercent := s.Config.DevFeePercent
			opPercent := s.Config.OperatorFeePercent
			cycleMins := s.Config.FeeCycleMinutes
			s.mu.Unlock()

			if cycleMins <= 0 {
				cycleMins = 100 // Default to 100 minutes if not set or invalid
			}
			cycleLength := cycleMins * 60

			feePercent := devPercent + opPercent
			if feePercent <= 0 {
				continue
			}

			feeSeconds := int((feePercent / 100.0) * float64(cycleLength))
			devSeconds := int((devPercent / 100.0) * float64(cycleLength))
			opSeconds := int((opPercent / 100.0) * float64(cycleLength))

			if s.cycleOffset == -1 {
				if feeSeconds >= cycleLength {
					s.cycleOffset = 0
				} else {
					nonFeeLength := cycleLength - feeSeconds
					// Ensure at least 20 minutes (1200s) grace period before fee triggers to establish PeakHash
					maxR := nonFeeLength - 1200
					if maxR < 1 {
						maxR = 1
					}
					R := rand.Intn(maxR)
					nowSec := int(time.Now().Unix())
					s.cycleOffset = (feeSeconds + R - (nowSec % cycleLength) + cycleLength) % cycleLength
				}
			}

			nowSec := int(time.Now().Unix())
			secondInCycle := (nowSec + s.cycleOffset) % cycleLength

			targetMode := FeeModeNone
			if secondInCycle < devSeconds {
				targetMode = FeeModeDev
			} else if secondInCycle < devSeconds+opSeconds {
				targetMode = FeeModeOperator
			}

			// Pre-warm 3 seconds before target mode switches
			preWarmSec := (secondInCycle + 3) % cycleLength
			preWarmMode := FeeModeNone
			if preWarmSec < devSeconds {
				preWarmMode = FeeModeDev
			} else if preWarmSec < devSeconds+opSeconds {
				preWarmMode = FeeModeOperator
			}

			s.mu.Lock()
			// Handle Pre-Warming
			if preWarmMode != s.CurrentFeeMode && preWarmMode != FeeModeNone && !s.IsPreWarmed {
				s.IsPreWarmed = true
				if preWarmMode == FeeModeDev {
					devWorker := s.Config.DevWorker
					if devWorker == "" {
						devWorker = "dev_worker"
					}
					go s.ConnectFee(s.Config.DevWallet, devWorker)
				} else if preWarmMode == FeeModeOperator {
					opWorker := s.Config.OperatorWorker
					if opWorker == "" {
						opWorker = "op_worker"
					}
					go s.ConnectFee(s.Config.OperatorWallet, opWorker)
				}
			}

			// Handle Actual Switch
			var extranonceToSend *ExtranonceData
			var jobToSend string
			var currentMinerConn net.Conn

			if targetMode != s.CurrentFeeMode {
				s.CurrentFeeMode = targetMode
				s.IsPreWarmed = false // Reset pre-warm flag
				currentMinerConn = s.MinerConn

				if targetMode == FeeModeNone {
					s.TargetState = "MAIN"
					s.State = "SWITCHING_TO_MAIN"
					go s.EndFee()
					if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
						if s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size {
							ident := s.GetMinerIdentifier()
							isSafe := s.IsBuggyAsic
							if s.Config.SafeMiners != "" && strings.Contains(s.Config.SafeMiners, ident) {
								isSafe = true
							}
							if !isSafe {
								extranonceToSend = s.MainExtranonce
								s.mu.Lock()
								s.LastExtranonceCmdTime = time.Now()
								s.mu.Unlock()
							}
						}
					}
					// Zero-latency job injection using Global Dispatcher
					cachedJob := GlobalDispatcher.GetJob(s.Config.PoolAddress)
					if cachedJob == "" {
					    cachedJob = s.LatestMainJob
					}
					if cachedJob != "" && currentMinerConn != nil {
						jobToSend = forceCleanJobs(cachedJob)
					}
				} else if targetMode == FeeModeDev || targetMode == FeeModeOperator {
					s.TargetState = "FEE"
					s.State = "SWITCHING_TO_FEE"
					if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
						if s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size {
							ident := s.GetMinerIdentifier()
							isSafe := s.IsBuggyAsic
							if s.Config.SafeMiners != "" && strings.Contains(s.Config.SafeMiners, ident) {
								isSafe = true
							}
							if !isSafe {
								extranonceToSend = s.FeeExtranonce
								s.mu.Lock()
								s.LastExtranonceCmdTime = time.Now()
								s.mu.Unlock()
							}
						}
					}
					// Zero-latency job injection for Fee
					cachedJob := s.LatestFeeJob
					if cachedJob != "" && currentMinerConn != nil {
						jobToSend = forceCleanJobs(cachedJob)
					}
				}
			}
			s.mu.Unlock()

			// Perform TCP socket writes outside of the mutex to prevent deadlocks
			if extranonceToSend != nil {
				s.sendExtranonce(extranonceToSend)
			}
			if jobToSend != "" && currentMinerConn != nil {
				fmt.Fprintf(currentMinerConn, "%s\n", jobToSend)
			}
		}
	}
}

func (s *Session) ConnectFee(wallet, worker string) {
	s.mu.Lock()
	if s.FeeConn != nil {
		s.mu.Unlock()
		return
	}
	if len(s.loginPackets) == 0 {
		// Wait until miner has authorized before connecting to fee pool
		s.TargetState = "MAIN"
		s.State = "MAIN"
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.LogGeneral("Initiating Smart Fee Routing...")

	// --- SMART ROUTING IDENTITY & FALLBACK ---
	// Determine Identity based on provided wallet input length
	isFeeSubAccount := len(wallet) < 20 && !strings.HasPrefix(wallet, "0x")
	isMinerSubAccount := len(s.MinerWallet) < 20 && !strings.HasPrefix(s.MinerWallet, "0x")

	host := s.Config.PoolAddress // 默认优先同池抽水
	s.SamePoolFeeActive = true

	// 如果当前抽水钱包是子账户，但矿工钱包是原生地址，直接跳过同池抽水回退到F2Pool，避免Auth失败长达数小时
	if isFeeSubAccount && !isMinerSubAccount {
		host = s.Config.FeePoolAddress
		s.SamePoolFeeActive = false
	}

	// 差异化回退逻辑
	if s.FeeAuthFailures > 0 {
		s.LogGeneral("[SmartRouting] Fallback triggered: Switching to F2Pool due to previous auth/share failures.")
		host = s.Config.FeePoolAddress
		s.SamePoolFeeActive = false
	}
	// -----------------------------------------

	s.LogGeneral("Connecting to Fee Pool: %s (Identity: %s)", host, wallet)

	// Create fee connection
	feeConn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		s.LogGeneral("Fee connection failed: %v", err)
		s.FeeAuthFailures++
		s.EndFee()
		return
	}
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(feeConn)
	}

	s.mu.Lock()
	s.FeeConn = feeConn
	s.mu.Unlock()

	// Replay login packets
	for _, pkt := range s.loginPackets {
		// Deep copy to not mutate original
		pktBytes, _ := json.Marshal(pkt)
		var mod map[string]interface{}
		_ = json.Unmarshal(pktBytes, &mod)

		method, _ := mod["method"].(string)
		if method == "mining.authorize" || method == "eth_submitLogin" || method == "login" {
			if params, ok := mod["params"].([]interface{}); ok && len(params) > 0 {
				if _, ok := params[0].(string); ok {
					// Always use wallet.worker format for maximum compatibility with F2Pool/Binance Pool
					mod["params"].([]interface{})[0] = fmt.Sprintf("%s.%s", wallet, worker)
					
					// If the protocol supports password, keep it as 'x' or the original password
					if len(params) > 1 {
						if pwd, isStr := params[1].(string); isStr && pwd == "" {
							mod["params"].([]interface{})[1] = "x"
						}
					}
				}
			} else if paramsMap, ok := mod["params"].(map[string]interface{}); ok {
				if _, ok := paramsMap["wallet"]; ok {
					paramsMap["wallet"] = wallet
				}
				if _, ok := paramsMap["worker"]; ok {
					paramsMap["worker"] = worker
				}
			}

			// Strip root-level worker field to prevent F2Pool from misinterpreting it as the account name
			delete(mod, "worker")

			// Inject fee fixed difficulty
			feeDiff := s.Config.FeeFixedDifficulty
			if feeDiff == "auto" {
				s.mu.Lock()
				curDiff := s.CurrentDiff
				s.mu.Unlock()
				if curDiff > 0 {
					feeDiff = fmt.Sprintf("d=%.0f", curDiff)
				} else {
					feeDiff = ""
				}
			}

			if feeDiff != "" {
				if params, ok := mod["params"].([]interface{}); ok && len(params) > 0 {
					if len(params) > 1 {
						params[1] = feeDiff
					} else {
						mod["params"] = append(params, feeDiff)
					}
				} else if paramsMap, ok := mod["params"].(map[string]interface{}); ok {
					paramsMap["pass"] = feeDiff
					paramsMap["password"] = feeDiff
				}
			}
		}
		modBytes, _ := json.Marshal(mod)
		fmt.Fprintf(feeConn, "%s\n", string(modBytes))
	}

	if s.Protocol == "ETH_PROXY" {
		getWorkPkt := `{"id": 0, "method": "eth_getWork", "params": []}` + "\n"
		_, _ = feeConn.Write([]byte(getWorkPkt))
	}

	// Read loop
	go func() {
		defer s.EndFee()
		scanner := bufio.NewScanner(feeConn)
		bufPtr := ScannerBufferPool.Get().(*[]byte)
		buf := (*bufPtr)[:0]
		defer ScannerBufferPool.Put(bufPtr)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			if s.Config.EnableDetailedLog {
				s.LogGeneral("[RAW FEE RX] %s", strings.TrimSpace(line))
			}

			var msg map[string]interface{}
			var isShareReply bool
			var isEthGetWorkReply bool
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				if s.Config.EnableAsic {
					s.mu.Lock()
					subId := s.SubscribeID
					s.mu.Unlock()
					if id, ok := msg["id"]; ok && id != nil && id == subId {
						if result, ok := msg["result"].([]interface{}); ok && len(result) > 2 {
							if en1, ok := result[1].(string); ok {
								if en2size, ok := result[2].(float64); ok {
									s.mu.Lock()
									en := &ExtranonceData{En1: en1, En2Size: int(en2size)}
									s.FeeExtranonce = en
									state := s.State
									mainEn := s.MainExtranonce
									s.mu.Unlock()
									if state == "FEE" || state == "SWITCHING_TO_FEE" {
										if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
											if mainEn == nil || mainEn.En2Size != en.En2Size {
												ident := s.GetMinerIdentifier()
												isSafe := s.IsBuggyAsic
												if s.Config.SafeMiners != "" && strings.Contains(s.Config.SafeMiners, ident) {
													isSafe = true
												}
												if !isSafe {
													s.sendExtranonce(en)
													s.mu.Lock()
													s.LastExtranonceCmdTime = time.Now()
													s.mu.Unlock()
												}
											}
										}
									}
								}
							}
						}
					}
					if method, ok := msg["method"].(string); ok && method == "mining.set_extranonce" {
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 1 {
							if en1, ok := params[0].(string); ok {
								if en2size, ok := params[1].(float64); ok {
									s.mu.Lock()
									en := &ExtranonceData{En1: en1, En2Size: int(en2size)}
									s.FeeExtranonce = en
									state := s.State
									mainEn := s.MainExtranonce
									s.mu.Unlock()
									if state == "FEE" || state == "SWITCHING_TO_FEE" {
										if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
											if mainEn == nil || mainEn.En2Size != en.En2Size {
												ident := s.GetMinerIdentifier()
												isSafe := s.IsBuggyAsic
												if s.Config.SafeMiners != "" && strings.Contains(s.Config.SafeMiners, ident) {
													isSafe = true
												}
												if !isSafe {
													s.sendExtranonce(en)
													s.mu.Lock()
													s.LastExtranonceCmdTime = time.Now()
													s.mu.Unlock()
												}
											}
										}
									}
								}
							}
						}
					}
				}

				if id, ok := msg["id"]; ok && id != nil {
					if origReq, isShareReply := s.pendingShares.LoadAndDelete(id); isShareReply {
						isReject := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							s.Stats.InvalidShares++
							isReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							s.Stats.InvalidShares++
							isReject = true
						} else if res, ok := msg["result"]; ok && res == true {
							s.Stats.FeeShares++
							s.Stats.ValidShares++
							s.mu.Lock()
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.mu.Unlock()
						}

						if isReject {
							if s.Config.EnableDetailedLog {
								s.LogError("[FEE] share rejected! Pool Response: %s | Original Request: %s", strings.TrimSpace(line), origReq)
							} else {
								s.LogError("[FEE] share rejected! %s", strings.TrimSpace(line))
							}
							
							if s.SamePoolFeeActive {
								s.FeeAuthFailures++
								if s.FeeAuthFailures >= 3 {
									s.LogGeneral("[SmartRouting] 3 consecutive share rejects. Triggering Fallback.")
									go func() {
										s.EndFee()
										s.ConnectFee(s.Config.DevWallet, s.Config.DevWorker)
									}()
									return // exit read loop
								}
							}
							
							s.mu.Lock()
							antiBan := s.Config.EnableAntiBan
							s.mu.Unlock()
							if antiBan {
								line = fmt.Sprintf(`{"id": %v, "result": true, "error": null}`, id)
							}
						} else {
							s.LogGeneral("[FEE] share accepted! [Diff: %.4f]", s.CurrentDiff)
						}
					} else {
						// NOT a share reply. Could be auth response or eth_getWork response.
						isAuthReject := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							isAuthReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							isAuthReject = true
						}
						
						if isAuthReject {
							s.LogGeneral("[SmartRouting] Fee Pool Auth/Generic Error: %v", line)
							if s.SamePoolFeeActive {
								// We are in same-pool fee mode and got rejected.
								s.FeeAuthFailures++
								// Force reconnect
								go func() {
									s.EndFee()
									// ConnectFee will automatically pick up the fallback logic since FeeAuthFailures > 0
									s.ConnectFee(s.Config.DevWallet, s.Config.DevWorker)
								}()
								return // exit read loop
							}
						}

						// Check for eth_getWork response
						if resArr, ok := msg["result"].([]interface{}); ok && len(resArr) >= 3 {
							if powHash, ok := resArr[0].(string); ok && strings.HasPrefix(powHash, "0x") {
								s.addJob(powHash, false) // false = Fee
								isEthGetWorkReply = true
								
								s.mu.Lock()
								s.LatestFeeJob = line
								s.mu.Unlock()
								
								// Target Hash Rewriting Optimization
								var finalTarget string
								if s.Config.EnableEthTargetRewrite {
									s.mu.Lock()
									targetHash := s.currentMainTargetHash
									s.mu.Unlock()
									if targetHash != "" {
										resArr[2] = targetHash
										finalTarget = targetHash
										msg["result"] = resArr
										if modBytes, err := json.Marshal(msg); err == nil {
											line = string(modBytes)
										}
									}
								}
								if finalTarget == "" {
									if th, ok := resArr[2].(string); ok {
										finalTarget = th
									}
								}
								if finalTarget != "" && strings.HasPrefix(finalTarget, "0x") {
									diffVal := parseEthProxyTargetToDiff(finalTarget)
									s.mu.Lock()
									s.CurrentDiff = diffVal
									s.mu.Unlock()
								}
							}
						}
					}
				} else if method, ok := msg["method"].(string); ok {
					if method == "mining.set_difficulty" {
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if diffFloat, ok := params[0].(float64); ok {
								s.mu.Lock()
								s.CurrentDiff = diffFloat
								s.mu.Unlock()
							}
						}
					} else if method == "mining.notify" {
						s.mu.Lock()
						s.LatestFeeJob = line
						s.mu.Unlock()
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if jobID, ok := params[0].(string); ok {
								s.addJob(jobID, false) // false = Fee
							}
						}
					}
				}
			}

			s.mu.Lock()
			state := s.State
			minerConn := s.MinerConn
			s.mu.Unlock()

			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if minerConn != nil {
					// Forward specific methods, share replies, and eth_getWork replies
					if isShareReply || isEthGetWorkReply {
						fmt.Fprintf(minerConn, "%s\n", line)
					} else if method, ok := msg["method"].(string); ok {
						if method == "mining.notify" || method == "mining.set_difficulty" || method == "mining.set_extranonce" || method == "eth_getWork" {
							fmt.Fprintf(minerConn, "%s\n", line)
						}
					}
				}
			}
		}
	}()
}

func (s *Session) EndFee() {
	s.mu.Lock()

	if s.State == "FEE" || s.State == "SWITCHING_TO_FEE" {
		s.State = "SWITCHING_TO_MAIN"
		s.TargetState = "MAIN"
	}

	connToClose := s.FeeConn
	s.mu.Unlock()

	if connToClose != nil {
		// Grace period: keep fee connection alive for 10 seconds to catch late shares
		go func(c net.Conn) {
			time.Sleep(10 * time.Second)
			c.Close()

			s.mu.Lock()
			// Only nil it if it hasn't been overwritten by a new fee cycle
			if s.FeeConn == c {
				s.FeeConn = nil
			}
			s.mu.Unlock()
		}(connToClose)
	}
}

type ExtranonceData struct {
	En1     string
	En2Size int
}

func (s *Session) sendExtranonce(extranonce *ExtranonceData) {
	s.mu.Lock()
	minerConn := s.MinerConn
	s.mu.Unlock()

	if extranonce == nil || minerConn == nil {
		return
	}
	msg := map[string]interface{}{
		"id":     nil,
		"method": "mining.set_extranonce",
		"params": []interface{}{extranonce.En1, extranonce.En2Size},
	}
	msgBytes, _ := json.Marshal(msg)
	
	// Write without holding the session mutex to prevent TCP block deadlocks
	// if the miner silently disconnects and the buffer fills up.
	fmt.Fprintf(minerConn, "%s\n", string(msgBytes))
}

func (s *Session) Watchdog() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.mu.Lock()
			lastShare := s.LastShareTime
			connAt := s.Stats.ConnectedAt
			lastExt := s.LastExtranonceCmdTime
			isBuggy := s.IsBuggyAsic
			enableAuto := s.Config.EnableAutoQuarantine
			s.mu.Unlock()

			now := time.Now()

			// Phase 1: 0 shares for 60s
			if enableAuto && !isBuggy && !lastExt.IsZero() && now.Sub(lastExt) > 60*time.Second {
				if lastShare.Before(lastExt) {
					s.mu.Lock()
					s.IsBuggyAsic = true
					s.mu.Unlock()
					ident := s.GetMinerIdentifier()
					s.LogGeneral("🤖 [AI-Quarantine] ASIC Hashboard Crash Detected (No shares 60s after command). Auto-Quarantining: %s", ident)
					_ = db.AddSafeMiner(s.Config.ListenPort, ident)

					s.mu.Lock()
					if s.Config.SafeMiners == "" {
						s.Config.SafeMiners = ident
					} else {
						s.Config.SafeMiners += "," + ident
					}
					s.mu.Unlock()
				}
			}

			// Phase 2: Hashrate Drop Detection
			uptime := now.Sub(connAt)
			if enableAuto && !isBuggy && uptime > 20*time.Minute && s.PeakHash > 0 && !lastExt.IsZero() && now.Sub(lastExt) < 30*time.Minute {
				if s.DisplayHash < s.PeakHash * 0.4 {
					s.mu.Lock()
					s.IsBuggyAsic = true
					s.mu.Unlock()
					ident := s.GetMinerIdentifier()
					s.LogGeneral("🤖 [AI-Quarantine] Severe Hashrate Drop Detected (Peak: %.2f, Now: %.2f). Auto-Quarantining and Force Resetting: %s", s.PeakHash, s.DisplayHash, ident)
					_ = db.AddSafeMiner(s.Config.ListenPort, ident)
					
					s.mu.Lock()
					if s.Config.SafeMiners == "" {
						s.Config.SafeMiners = ident
					} else {
						s.Config.SafeMiners += "," + ident
					}
					s.mu.Unlock()
					
					s.Close() // Force physical reset to trigger 150T recovery
					return
				}
			}

			if now.Sub(lastShare) > 10*time.Minute && now.Sub(connAt) > 5*time.Minute {
				log.Printf("[Watchdog] Miner %s timed out (no shares for 10 mins). Force closing.", s.ID)
				s.Close()
				return
			}
		}
	}
}

func (s *Session) GetMinerIdentifier() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.MinerWallet != "" && s.MinerWorker != "" {
		return fmt.Sprintf("%s.%s", s.MinerWallet, s.MinerWorker)
	}
	if s.MinerWorker != "" {
		return s.MinerWorker
	}
	if s.MinerWallet != "" {
		return s.MinerWallet
	}
	return s.MinerConn.RemoteAddr().String()
}
