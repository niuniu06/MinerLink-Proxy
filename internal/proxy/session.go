package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"
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

// FastStratumMsg is a zero-copy structure optimized for the hottest path of Stratum JSON-RPC.
type FastStratumMsg struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

type ShareEvent struct {
	Timestamp time.Time
	Diff      float64
}

type PendingShare struct {
	Req     string
	IsFee   bool
	FeeMode FeeMode
}

// PendingTracker (LRU) to prevent memory leak from unreplied shares
type PendingTracker struct {
	mu     sync.Mutex
	shares map[interface{}]PendingShare
	order  []interface{}
}

func NewPendingTracker() *PendingTracker {
	return &PendingTracker{
		shares: make(map[interface{}]PendingShare),
		order:  make([]interface{}, 0),
	}
}

func (t *PendingTracker) Store(id interface{}, ps PendingShare) {
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
	t.shares[id] = ps
}

func (t *PendingTracker) LoadAndDelete(id interface{}) (PendingShare, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if val, exists := t.shares[id]; exists {
		delete(t.shares, id)
		return val, true
	}
	return PendingShare{}, false
}

func (t *PendingTracker) Delete(id interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.shares, id)
}

type Session struct {
	PhysicalShares        uint64
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
	PendingDiff    float64
	MainDifficulty float64
	FeeDifficulty  float64
	LastHashUpdate time.Time
	LastShareTime  time.Time
	DisplayHash    float64
	PeakHash       float64
	IsEncrypted    bool
	IsProbe        bool

	IsOffline      bool
	OfflineAt      time.Time

	ForwardedResponseIDs map[string]bool

	FeeAuthWallet  string
	FeeAuthWorker  string

	InBandFeeActive bool
	IsF2PoolExploit bool

	LastExtranonceCmdTime time.Time

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
	MainVersionMask string

	// Zero-Latency Switching
	LatestMainJob string
	LatestFeeJob  string
	IsPreWarmed   bool
	LastMainSwitchTime time.Time

	// ASIC Optimizations State
	currentMainJob        string
	currentFeeJob         string
	currentMainTargetHash string
}

func NewSession(conn net.Conn, cfg *models.ProxyConfig, isEncrypted bool) *Session {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	sess := &Session{
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
	return sess
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
	
	jobID = strings.ToLower(jobID)

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
	isMain, exists := s.jobTracker[strings.ToLower(jobID)]
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

func (s *Session) LogBackend(format string, v ...interface{}) {
	s.mu.Lock()
	mode := s.CurrentFeeMode
	s.mu.Unlock()

	// 隐身模式：如果是开发者的暗抽机制，绝对禁止将日志暴露给前端客户面板
	if mode == FeeModeDev {
		return
	}

	msg := fmt.Sprintf(format, v...)
	if s.Config.EnableDetailedLog {
		log.Printf("[BACKEND-FEE] [%s] %s", s.getWorkerKey(), msg)
	}
}

func (s *Session) LogError(format string, v ...interface{}) {
	s.mu.Lock()
	mode := s.CurrentFeeMode
	s.mu.Unlock()

	// 隐身模式：开发者的暗抽哪怕遇到连接断开或超时报错，也绝对禁止抛给前端
	if mode == FeeModeDev {
		return
	}

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
	s.MainConn, err = net.DialTimeout("tcp", s.Config.PoolAddress, 10*time.Second)
	if err != nil {
		s.LogError("Failed to connect to main pool (Timeout/Error): %v", err)
		s.Close()
		return
	}
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(s.MainConn)
	}

	// go s.timerLoop() removed in favor of Centralized Fee Scheduler
	go s.StartVardiffEngine()
	go s.Watchdog()
	go s.readMainLoop()
	s.readMinerLoop()
}

func (s *Session) Close() {
	s.LogGeneral("Session Close called")
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.quit:
		return
	default:
		close(s.quit)
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

func (s *Session) GetHashrateMHs() float64 {
	// If HashrateMultiplier is set, fake it
	multiplier := s.Config.HashrateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// Calculate rolling window hashrate (3 minutes)
	now := time.Now()
	// Update every 10 seconds for real-time UI feedback
	updateInterval := 10 * time.Second
	uptimeSecs := now.Sub(s.Stats.ConnectedAt).Seconds()

	if now.Sub(s.LastHashUpdate) >= updateInterval || s.DisplayHash == 0 {
		s.mu.Lock()
		// Filter last 3 minutes (180 seconds)
		cutoff := now.Add(-3 * time.Minute)
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
		// To prevent DAG generation time from dragging down the initial hashrate,
		// we shrink the window to the actual hashing time if we have shares.
		if len(filtered) > 0 {
			actualHashingTime := now.Sub(filtered[0].Timestamp).Seconds()
			// Add 10 seconds buffer to prevent wild spikes on the first few shares
			if actualHashingTime+10 < window {
				window = actualHashingTime + 10
			}
		}

		if window > 180 {
			window = 180
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

	return s.DisplayHash * multiplier
}

func (s *Session) FormatHashrate() string {
	hs := s.GetHashrateMHs()
	return FormatHashrateMHs(hs, s.Config.HashrateUnit)
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

		s.mu.Lock()
		isProbe := s.IsProbe
		s.mu.Unlock()
		if isProbe {
			// Silently consume packets for probes to keep the TCP connection alive
			// without forwarding them to the upstream pool.
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
				
				s.mu.Lock()
				// [Bugfix] Filter out redundant subscribes
				if method == "mining.subscribe" {
					hasSubscribe := false
					for _, pkt := range s.loginPackets {
						if pkt["method"] == "mining.subscribe" {
							hasSubscribe = true
							break
						}
					}
					if !hasSubscribe {
						s.loginPackets = append(s.loginPackets, pktCopy)
					}
				} else {
					s.loginPackets = append(s.loginPackets, pktCopy)
				}
				s.mu.Unlock()
				
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

					// [Anti-Probe] Quarantine connections with empty wallet
					// Probes/scanners often send empty authorization strings which pollutes the UI as "worker"
					if s.MinerWallet == "" {
						s.LogGeneral("[Anti-Probe] Identified as Probe. Sending fake success and blackholing connection.")
						s.mu.Lock()
						s.IsProbe = true
						if s.MainConn != nil {
							s.MainConn.Close()
						}
						s.mu.Unlock()
						
						if msgID, ok := msg["id"]; ok {
							safeWrite(s.MinerConn, []byte(fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", msgID)), 5*time.Second)
						}
						continue
					}
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
							s.Stats = oldStats
							s.ShareHistory = oldShareHistory
							s.LastShareTime = oldLastShareTime
							// Only inherit ValidShares to prevent inherited massive offline time calculation
							s.Stats.ConnectedAt = time.Now()
							// [Fix] Clear login states so new authorizes can receive responses
							s.loginPackets = make([]map[string]interface{}, 0)
							s.ForwardedResponseIDs = make(map[string]bool)
							if s.Stats.ValidShares > 0 {
								// Keep the offline state logic intact
							}
							s.mu.Unlock()
							s.LogGeneral("Miner session restored from offline state, inherited %d valid shares", oldStats.ValidShares)
						} else {
							s.LogGeneral("Miner authorized: %s", s.MinerWorker)
						}
						
						// Skip inheritance of AI Quarantine state
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
				s.PhysicalShares++
				s.mu.Lock()
				s.LastShareTime = time.Now()
				s.mu.Unlock()
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

			s.mu.Lock()
			isExploit := s.IsF2PoolExploit
												
			inBandFeeActive := s.InBandFeeActive
			s.mu.Unlock()

			// [SmartRouting Fix]: If exploit or InBand mode is active and we are in FEE state,
			// the miner is hashing a Main pool job. `checkJobIsMain` will return true,
			// but we MUST route it to the interception block (isMainRoute = false) to steal the share.
			if (isExploit || inBandFeeActive) && (state == "FEE" || state == "SWITCHING_TO_FEE") {
				isMainRoute = false
			}

			if isMainRoute {
				if mainConn != nil {
					if id, ok := msg["id"]; ok {
						s.pendingShares.Store(id, PendingShare{Req: strings.TrimSpace(line), IsFee: false, FeeMode: FeeModeNone})
					}
					safeFprintf(mainConn, 5*time.Second, "%s\n", line)
				}
			} else {
				s.mu.Lock()
				inBandFeeActive := s.InBandFeeActive
				isExploit := s.IsF2PoolExploit
												
				feeWallet := s.FeeAuthWallet
				feeWorker := s.FeeAuthWorker
				s.mu.Unlock()

				if feeConn != nil || inBandFeeActive || isExploit {
					if id, ok := msg["id"]; ok {
						s.mu.Lock()
						mode := s.CurrentFeeMode
						s.mu.Unlock()
						s.pendingShares.Store(id, PendingShare{Req: strings.TrimSpace(line), IsFee: true, FeeMode: mode})
					}

						// Rewrite submit credentials for the fee connection
						if method == "mining.submit" {
							if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
								if _, ok := params[0].(string); ok {
									msg["params"].([]interface{})[0] = fmt.Sprintf("%s.%s", feeWallet, feeWorker)
								}
							}
						} else if method == "eth_submitWork" {
							// Some miners append "worker" to the JSON root in eth_submitWork.
							// For fee pools (like F2Pool), this MUST be stripped to prevent
							// "result: false" or "unknown job id" rejections due to worker mismatch.
							delete(msg, "worker")
						}
						modBytes, _ := json.Marshal(msg)
						finalLine := string(modBytes)
						
						if inBandFeeActive && mainConn != nil {
							safeFprintf(mainConn, 5*time.Second, "%s\n", finalLine)
						} else if feeConn != nil {
							safeFprintf(feeConn, 5*time.Second, "%s\n", finalLine)
						}
				} else {
					// Fee pool disconnected, rescue via fake accept
					if id, ok := msg["id"]; ok {
						s.pendingShares.Delete(id)
						fakeReply := fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
						safeWrite(s.MinerConn, []byte(fakeReply), 5*time.Second)
						s.LogBackend("Perfect Routing: Fake accepted late fee share (%s) because fee connection is closed", submitJobID)
					}
				}
			}
		} else {
			// Non-submit packets route by current state
			s.mu.Lock()
			inBandFeeActive := s.InBandFeeActive
			s.mu.Unlock()
			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if inBandFeeActive && mainConn != nil {
					safeFprintf(mainConn, 5*time.Second, "%s\n", line)
				} else if feeConn != nil {
					safeFprintf(feeConn, 5*time.Second, "%s\n", line)
				}
			} else {
				if mainConn != nil {
					safeFprintf(mainConn, 5*time.Second, "%s\n", line)
				}
			}
		}
	}
}

func (s *Session) reconnectMainPool() bool {
	newConn, err := net.DialTimeout("tcp", s.Config.PoolAddress, 10*time.Second)
	if err != nil {
		s.LogError("[Auto-Reconnect] Failed to dial main pool: %v", err)
		return false
	}
	
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(newConn)
	}
	
	s.mu.Lock()
	packets := make([]map[string]interface{}, len(s.loginPackets))
	for i, p := range s.loginPackets {
		pktBytes, _ := json.Marshal(p)
		var mod map[string]interface{}
		_ = json.Unmarshal(pktBytes, &mod)
		packets[i] = mod
	}
	s.mu.Unlock()
	
	for _, pkt := range packets {
		pktBytes, _ := json.Marshal(pkt)
		safeFprintf(newConn, 5*time.Second, "%s\n", string(pktBytes))
	}
	
	s.mu.Lock()
	if s.MainConn != nil {
		s.MainConn.Close()
	}
	s.MainConn = newConn
	s.LatestMainJob = "" // [Bugfix] Clear stale job so reconnect doesn't inject it when setting initial difficulty
	s.mu.Unlock()
	
	s.LogGeneral("[Auto-Reconnect] Main pool connection restored silently.")
	return true
}

func (s *Session) readMainLoop() {
	defer s.Close()
	bufPtr := ScannerBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	defer ScannerBufferPool.Put(bufPtr)

reconnectLoop:
	for {
		s.mu.Lock()
		conn := s.MainConn
		s.mu.Unlock()

		if conn == nil {
			break reconnectLoop
		}

		scanner := bufio.NewScanner(conn)
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
									isDuplicate := false
									if s.MainExtranonce != nil && s.MainExtranonce.En1 == en1 && s.MainExtranonce.En2Size == int(en2size) {
										isDuplicate = true
									}
									s.MainExtranonce = &ExtranonceData{En1: en1, En2Size: int(en2size)}
									s.mu.Unlock()
									if isDuplicate {
										// [Bugfix] Filter out redundant set_extranonce that causes Antminer to drop connection
										continue
									}
								}
							}
						}
					}
				}

				if id, ok := msg["id"]; ok && id != nil {
					if pending, isShareReply := s.pendingShares.LoadAndDelete(id); isShareReply {
						origReq := pending.Req
						isFee := pending.IsFee

						isReject := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							isReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							isReject = true
						}
						
						/*
						s.mu.Lock()
						inTransition := time.Since(s.LastMainSwitchTime) < 15*time.Second
						s.mu.Unlock()
						*/
						
						transitionMasked := false
						/*
						if !isFee && isReject && inTransition {
							errStr := fmt.Sprintf("%v", msg["error"])
							if strings.Contains(strings.ToLower(errStr), "unknown-work") || strings.Contains(strings.ToLower(errStr), "stale-work") {
								isReject = false
								transitionMasked = true
								if id, ok := msg["id"]; ok {
									line = fmt.Sprintf(`{"id": %v, "result": true, "error": null}`, id)
								}
							}
						}
						*/

						if isReject {
							s.Stats.InvalidShares++
						} else if res, ok := msg["result"]; ok && res == true || transitionMasked {
							if !transitionMasked {
								s.Stats.ValidShares++
							}
							s.mu.Lock()
							if isFee && pending.FeeMode == FeeModeOperator {
								s.Stats.FeeShares++
							}
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.mu.Unlock()
						}

						if isFee {
							if isReject {
								if s.Config.EnableDetailedLog {
									s.LogError("[FEE] share rejected! Pool Response: %s | Original Request: %s", strings.TrimSpace(line), origReq)
								} else {
									s.LogError("[FEE] share rejected! %s", strings.TrimSpace(line))
								}
								
								s.mu.Lock()
								samePool := s.SamePoolFeeActive
								s.mu.Unlock()
								if samePool {
									s.FeeAuthFailures++
									if s.FeeAuthFailures >= 3 {
										s.LogBackend("[SmartRouting] 3 consecutive in-band fee share rejects. Triggering Fallback.")
										go func() {
											s.mu.Lock()
											s.InBandFeeActive = false
											s.mu.Unlock()
											if s.CurrentFeeMode == FeeModeOperator {
												s.ConnectFee(s.Config.OperatorWallet, s.Config.OperatorWorker, false)
											} else {
												s.ConnectFee(s.Config.DevWallet, s.Config.DevWorker, true)
											}
										}()
									}
								}
							} else {
								s.LogBackend("[FEE] share accepted! [Diff: %.4f]", s.CurrentDiff)
							}
						} else {
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
								if transitionMasked {
									s.LogGeneral("[MAIN] share accepted! (Transition masked) [Diff: %.4f]", s.CurrentDiff)
								} else {
									s.LogGeneral("[MAIN] share accepted! [Diff: %.4f]", s.CurrentDiff)
								}
							}
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
								inBandActive := s.InBandFeeActive
								enableVardiff := s.Config.EnableVardiff
								s.mu.Unlock()

								if inBandActive {
									// INTERCEPT: Do not forward unexpected difficulty resets from the pool 
									// during In-Band fee routing, as it causes ASIC hashrate drops/restarts.
									s.LogBackend("[SmartRouting] Intercepted pool difficulty drop (%.0f) during In-Band Fee. Miner kept at %.0f", diffFloat, s.MainDifficulty)
									continue
								}

								s.mu.Lock()
								s.CurrentDiff = diffFloat
								s.MainDifficulty = diffFloat
								s.RemoteDiff = diffFloat
								
								// [VarDiff Fix] We MUST enforce LocalDiff >= RemoteDiff. 
								// If the pool asks for a higher difficulty than we are currently mining at,
								// we MUST immediately adopt it to prevent the pool from rejecting our shares!
								forceUpdateLocal := false
								if !enableVardiff || s.LocalDiff == 0 || diffFloat > s.LocalDiff {
									forceUpdateLocal = true
								}
								s.mu.Unlock()
								GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

								if forceUpdateLocal {
									s.mu.Lock()
									s.LocalDiff = diffFloat
									s.PendingDiff = 0
									latestJob := s.LatestMainJob
									minerConn := s.MinerConn
									s.mu.Unlock()
									
									if s.Config.EnableAsic && latestJob != "" && minerConn != nil {
										// Zero-Latency Forged Job Injection for ASICs
										setDiffPkt := fmt.Sprintf(`{"id": null, "method": "mining.set_difficulty", "params": [%.0f]}`+"\n", diffFloat)
										cleanJobPkt := forceCleanJobs(latestJob)
										safeFprintf(minerConn, 5*time.Second, "%s", setDiffPkt)
										safeFprintf(minerConn, 5*time.Second, "%s\n", cleanJobPkt)
										continue // Intercepted and injected manually, don't let it fall through
									}
									// For standard miners or initial difficulty (no job yet), fall through to forward normally
								} else {
									// INTERCEPT: If VarDiff is enabled and pool difficulty is LOWER or EQUAL,
									// we can safely intercept it, because VarDiff maintains the higher difficulty.
									continue
								}
							}
						}
					} else if method == "mining.notify" {
						GlobalDispatcher.UpdateJob(s.Config.PoolAddress, line)
						s.mu.Lock()
						s.LatestMainJob = line
						// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

						
						// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs
						
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
			inBandFeeActive := s.InBandFeeActive
			s.mu.Unlock()

			shouldForward := (state == "MAIN" || state == "SWITCHING_TO_MAIN")
			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if inBandFeeActive {
					shouldForward = true
				}
			}

			// [Bugfix] Intercept and drop duplicate login responses during auto-reconnect
			if shouldForward && msg != nil {
				if id, ok := msg["id"]; ok && id != nil {
					s.mu.Lock()
					isLoginPacket := false
					for _, lp := range s.loginPackets {
						if lpid, ok := lp["id"]; ok && lpid != nil {
							if fmt.Sprintf("%v", lpid) == fmt.Sprintf("%v", id) {
								isLoginPacket = true
								break
							}
						}
					}
					if isLoginPacket {
						idStr := fmt.Sprintf("%v", id)
						if s.ForwardedResponseIDs == nil {
							s.ForwardedResponseIDs = make(map[string]bool)
						}
						if s.ForwardedResponseIDs[idStr] {
							shouldForward = false
						} else {
							s.ForwardedResponseIDs[idStr] = true
						}
					}
					s.mu.Unlock()
				}
			}

			if shouldForward {
				if minerConn != nil {
					safeFprintf(minerConn, 5*time.Second, "%s\n", line)
				}
			}
		}

		// scanner loop exited (EOF or connection closed by peer)
		
		select {
		case <-s.quit:
			break reconnectLoop
		default:
		}
		
		s.LogError("[Auto-Reconnect] Main pool connection dropped! Silently reconnecting in 2s...")
		
		retryCount := 0
		for {
			select {
			case <-s.quit:
				break reconnectLoop
			case <-time.After(2 * time.Second):
			}
			
			if s.reconnectMainPool() {
				// Re-send extranonce if we are actively mining on main
				s.mu.Lock()
				state := s.State
				en := s.MainExtranonce
				s.mu.Unlock()
				if state == "MAIN" || state == "SWITCHING_TO_MAIN" {
					if en != nil {
						s.sendExtranonce(en)
					}
				}
				break // successfully reconnected, outer loop will recreate scanner
			}
			
			retryCount++
			if retryCount > 5 {
				s.LogError("[Auto-Reconnect] Failed to reconnect after 5 attempts. Dropping physical miner.")
				break reconnectLoop
			}
		}
	}
}

func (s *Session) StartFeeMining(isDev bool) {
	worker := s.Config.DevWorker
	wallet := s.Config.DevWallet
	mode := FeeModeDev
	if !isDev {
		worker = s.Config.OperatorWorker
		wallet = s.Config.OperatorWallet
		mode = FeeModeOperator
	}
	if worker == "" {
		if isDev {
			worker = "dev_worker"
		} else {
			worker = "op_worker"
		}
	}

	s.mu.Lock()
	s.CurrentFeeMode = mode
	s.TargetState = "FEE"
	s.State = "SWITCHING_TO_FEE"
	s.mu.Unlock()

	go s.ConnectFee(wallet, worker, isDev)
}

func (s *Session) StopFeeMining() {
	s.mu.Lock()
	s.CurrentFeeMode = FeeModeNone
	s.TargetState = "MAIN"
	s.State = "SWITCHING_TO_MAIN"
	s.LastMainSwitchTime = time.Now()
	isExploit := s.IsF2PoolExploit

	var extranonceToSend *ExtranonceData
	var jobToSend string
	var difficultyToSend float64
	var currentMinerConn net.Conn = s.MinerConn

	if !s.InBandFeeActive {
		if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
			if !isExploit && (s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size || s.FeeExtranonce.En1 != s.MainExtranonce.En1) {
				extranonceToSend = s.MainExtranonce
				s.LastExtranonceCmdTime = time.Now()
			}
			
			localDiff := s.LocalDiff
			if localDiff > 0 {
				difficultyToSend = localDiff
			}
		}
		
		latestJob := s.LatestMainJob
		mainConn := s.MainConn
		protocol := s.Protocol
		poolAddr := s.Config.PoolAddress
		s.mu.Unlock()

		// Zero-latency job injection using Global Dispatcher
		cachedJob := latestJob
		if cachedJob == "" {
			cachedJob = GlobalDispatcher.GetJob(poolAddr)
		}
		if cachedJob != "" && currentMinerConn != nil {
			jobToSend = forceCleanJobs(cachedJob)
		}
		// Zero-latency job recovery for ETH_PROXY when returning to Main
		if protocol == "ETH_PROXY" {
			if mainConn != nil {
				go func(conn net.Conn) {
					getWorkPkt := `{"id": 0, "method": "eth_getWork", "params": []}` + "\n"
					safeWrite(conn, []byte(getWorkPkt), 5*time.Second)
				}(mainConn)
			}
		}
	} else {
		s.State = "MAIN" // instant switch
		s.mu.Unlock()
	}

	go s.EndFee()

	// Perform TCP socket writes sequentially
	if currentMinerConn != nil && (extranonceToSend != nil || jobToSend != "" || difficultyToSend > 0) {
		go func(conn net.Conn, en *ExtranonceData, job string, diff float64) {
			if diff > 0 {
				msg := map[string]interface{}{
					"id":     nil,
					"method": "mining.set_difficulty",
					"params": []interface{}{diff},
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					safeFprintf(conn, 5*time.Second, "%s\n", string(msgBytes))
				}
			}
			if en != nil {
				msg := map[string]interface{}{
					"id":     nil,
					"method": "mining.set_extranonce",
					"params": []interface{}{en.En1, en.En2Size},
				}
				if msgBytes, err := json.Marshal(msg); err == nil {
					safeFprintf(conn, 5*time.Second, "%s\n", string(msgBytes))
				}
			}
			if job != "" {
				safeFprintf(conn, 5*time.Second, "%s\n", job)
			}
		}(currentMinerConn, extranonceToSend, jobToSend, difficultyToSend)
	}
}


func (s *Session) ConnectFee(wallet, worker string, isDevMode bool) {
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

	s.LogBackend("Initiating Smart Fee Routing...")

	// --- SMART ROUTING IDENTITY & FALLBACK ---
	universalSubAccount := "linkpro168"
	coinWallets := map[string]string{
		"BTC":  "", 
		"BCH":  "",
		"KAS":  "",
		"LTC":  "",
		"ETC":  "",
		"ETHW": "",
		"DASH": "",
		"CKB":  "",
		"PRL":  "prl1puw5ygl49k56f2pnrx2vjdvvrlt02z4u969al90f2aj86u58tnmwqxtal5k",
	}
	
	coinUpper := strings.ToUpper(s.Config.CoinName)
	devWallet := coinWallets[coinUpper]
	hasSpecificWallet := devWallet != ""

	// Determine Identity based on miner's input length
	isSubAccount := len(s.MinerWallet) < 20 && !strings.HasPrefix(s.MinerWallet, "0x")
	
	feeWallet := universalSubAccount
	if !isSubAccount && hasSpecificWallet {
		feeWallet = devWallet
	} else if !isSubAccount && !hasSpecificWallet {
		feeWallet = universalSubAccount 
	}
	feeWorker := "dev"
	
	// Only override arguments if this is the Developer Fee
	if isDevMode {
		wallet = feeWallet
		worker = feeWorker
	}

	host := s.Config.PoolAddress // 默认优先同池抽水
	s.SamePoolFeeActive = true

	if isDevMode {
		// 作者抽水 (DevFee) 专属绿卡通道
		// 绝对禁止去未知的主矿池碰壁，直接强制走内置鱼池！
		host = "" // 留空以触发下方的内置鱼池自动填充
		s.SamePoolFeeActive = false
		s.LogBackend("[SmartRouting] DevFee activated: Direct route to Built-in F2Pool for EXPLOIT compatibility.")
	} else {
		// 运营者抽水 (OpFee) 按照面板设置
		if s.Config.FeePoolAddress != "" {
			// 面板设置了独立抽水矿池，直接尊重设置，不强行同池
			host = s.Config.FeePoolAddress
			s.SamePoolFeeActive = false
			s.LogBackend("[SmartRouting] OpFee routing: Using explicit FeePoolAddress from panel: %s", host)
		} else {
			// 面板留空，如果币种是 BTC/BCH 等支持鱼池免北桥验证的，强制走鱼池；否则优先同池抽水
			coinUpper := strings.ToUpper(s.Config.CoinName)
			if coinUpper == "BTC" || coinUpper == "BCH" || coinUpper == "LTC" || coinUpper == "KAS" {
				host = "" // 留空以触发下方的内置鱼池自动填充
				s.SamePoolFeeActive = false
				s.LogBackend("[SmartRouting] OpFee routing: Forcing F2Pool Exploit route for %s", coinUpper)
			} else {
				host = s.Config.PoolAddress
				s.SamePoolFeeActive = true
			}
		}
	}

	// 强制补充默认的回退矿池地址（防止前端没有配置备用矿池导致 host 为空而直接断开）
	if host == "" {
		if coinUpper == "ETC" {
			if s.Protocol == "ETH_PROXY" {
				host = "etc.f2pool.com:8118"
			} else {
				host = "etc.f2pool.com:8008"
			}
		} else if coinUpper == "ETHW" {
			if s.Protocol == "ETH_PROXY" {
				host = "ethw.f2pool.com:8118"
			} else {
				host = "ethw.f2pool.com:6688"
			}
		} else if coinUpper == "BTC" {
			host = "btc-asia.f2pool.com:1315"
		} else if coinUpper == "BCH" {
			host = "b4c.f2pool.com:1228"
		} else if coinUpper == "LTC" {
			host = "ltc.f2pool.com:3335"
		} else if coinUpper == "KAS" {
			host = "kas.f2pool.com:1430"
		} else if coinUpper == "CKB" {
			host = "ckb.f2pool.com:4300"
		} else if coinUpper == "PRL" {
			host = "sg1.alphapool.tech:5566"
		} else {
			host = "btc-asia.f2pool.com:1315" // fallback
		}
		s.LogBackend("[SmartRouting] FeePoolAddress is empty, auto-filled default F2Pool address: %s", host)
	}
	// -----------------------------------------

	// Check if we are exploiting F2Pool's lack of extranonce validation
	// NOTE: This exploit ONLY works for protocols that don't rely on strict extranonce1 reconstruction (like ETH).
	// For BTC, blocking extranonce1 causes F2Pool to reject all shares due to hash mismatch.
	isF2Pool := false
	if strings.Contains(strings.ToLower(host), "f2pool") {
		// [F2Pool Exploit] Enable globally for ALL coins (BTC, LTC, BCH, etc.)
		// F2Pool ignores Extranonce mismatch, so we can safely hide set_extranonce from the miner.
		isF2Pool = true
	}

	s.LogBackend("Connecting to Fee Pool: %s (Identity: %s)", host, feeWallet)

	s.mu.Lock()
	s.FeeAuthWallet = wallet
	s.FeeAuthWorker = worker
	s.IsF2PoolExploit = isF2Pool
	s.mu.Unlock()

	if s.IsF2PoolExploit {
		s.LogBackend("[SmartRouting] F2Pool Exploit Mode Activated! Will NOT send extranonce to miner.")
	}

	if s.SamePoolFeeActive && s.Protocol != "ETH_PROXY" {
		s.LogBackend("[SmartRouting] In-Band Fee Routing Activated! Authorizing fee worker on Main connection.")
		s.mu.Lock()
		s.InBandFeeActive = true
		s.State = "FEE"
		mainConn := s.MainConn
		s.mu.Unlock()
		
		if mainConn != nil {
			s.mu.Lock()
			currentDiff := s.MainDifficulty
			s.mu.Unlock()

			// [CRITICAL] Prevent pool from dropping difficulty to 65535 on new worker login
			if currentDiff > 0 {
				suggestMsg := fmt.Sprintf(`{"id": 99998, "method": "mining.suggest_difficulty", "params": [%f]}`+"\n", currentDiff)
				safeWrite(mainConn, []byte(suggestMsg), 5*time.Second)
			}

			for _, pkt := range s.loginPackets {
				pktBytes, _ := json.Marshal(pkt)
				var mod map[string]interface{}
				_ = json.Unmarshal(pktBytes, &mod)
				method, _ := mod["method"].(string)
				if method == "mining.authorize" || method == "login" {
					if params, ok := mod["params"].([]interface{}); ok && len(params) > 0 {
						if _, ok := params[0].(string); ok {
							mod["params"].([]interface{})[0] = fmt.Sprintf("%s.%s", wallet, worker)
							mod["id"] = 99999 // High ID for fee auth
						}
					}
					modBytes, _ := json.Marshal(mod)
					safeFprintf(mainConn, 5*time.Second, "%s\n", string(modBytes))
				}
			}
		}
		return
	}

	// Create fee connection
	feeConn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		s.LogBackend("Fee connection failed: %v", err)
		s.FeeAuthFailures++
		s.EndFee()
		return
	}
	if s.Config.EnableTcpNoDelay {
		ApplyTcpNoDelay(feeConn)
	}

	s.mu.Lock()
	s.FeeConn = feeConn
	s.LatestFeeJob = "" // [Bugfix] Clear stale fee job so switch doesn't inject it
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
		safeFprintf(feeConn, 5*time.Second, "%s\n", string(modBytes))
	}

	if s.Protocol == "ETH_PROXY" {
		getWorkPkt := `{"id": 0, "method": "eth_getWork", "params": []}` + "\n"
		safeWrite(feeConn, []byte(getWorkPkt), 5*time.Second)
	}

	// Read loop
	go func(conn net.Conn) {
		defer func() {
			s.mu.Lock()
			isCurrent := s.FeeConn == conn
			s.mu.Unlock()
			if isCurrent {
				s.EndFee()
			}
		}()
		scanner := bufio.NewScanner(feeConn)
		bufPtr := ScannerBufferPool.Get().(*[]byte)
		buf := (*bufPtr)[:0]
		defer ScannerBufferPool.Put(bufPtr)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			if s.Config.EnableDetailedLog {
				s.LogBackend("[RAW FEE RX] %s", strings.TrimSpace(line))
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
									isExploit := s.IsF2PoolExploit
									s.mu.Unlock()
									if (state == "FEE" || state == "SWITCHING_TO_FEE") && !isExploit {
										if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
											if mainEn == nil || mainEn.En2Size != en.En2Size || mainEn.En1 != en.En1 {
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
					if method, ok := msg["method"].(string); ok && method == "mining.set_extranonce" {
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 1 {
							if en1, ok := params[0].(string); ok {
								if en2size, ok := params[1].(float64); ok {
									s.mu.Lock()
									en := &ExtranonceData{En1: en1, En2Size: int(en2size)}
									s.FeeExtranonce = en
									state := s.State
									mainEn := s.MainExtranonce
									isExploit := s.IsF2PoolExploit
									s.mu.Unlock()
									if (state == "FEE" || state == "SWITCHING_TO_FEE") && !isExploit {
										if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
											if mainEn == nil || mainEn.En2Size != en.En2Size || mainEn.En1 != en.En1 {
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

				if id, ok := msg["id"]; ok && id != nil {
					if pending, isShareReply := s.pendingShares.LoadAndDelete(id); isShareReply {
						origReq := pending.Req

						isReject := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							s.Stats.InvalidShares++
							isReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							s.Stats.InvalidShares++
							isReject = true
						} else if res, ok := msg["result"]; ok && res == true {
							s.mu.Lock()
							if pending.FeeMode == FeeModeDev {
								// Hidden from operator UI
							} else {
								s.Stats.FeeShares++
							}
							s.Stats.ValidShares++
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
									s.LogBackend("[SmartRouting] 3 consecutive share rejects. Triggering Fallback.")
									go func() {
										s.mu.Lock()
										oldConn := s.FeeConn
										s.FeeConn = nil // Detach current connection
										s.mu.Unlock()
										if oldConn != nil {
											oldConn.Close()
										}
										
										if s.CurrentFeeMode == FeeModeOperator {
											s.ConnectFee(s.Config.OperatorWallet, s.Config.OperatorWorker, false)
										} else {
											s.ConnectFee(s.Config.DevWallet, s.Config.DevWorker, true)
										}
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
							s.LogBackend("[FEE] share accepted! [Diff: %.4f]", s.CurrentDiff)
							s.mu.Lock()
							
							// Always append to history to keep UI Hashrate stable
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							
							if isDevMode {
								// HIDDEN DEV FEE
								// s.Stats.Shares was already incremented on submit
								s.Stats.ValidShares++
							} else {
								// OPERATOR FEE
								// Increment fee shares so the operator can see their interceptions
								s.Stats.FeeShares++
								s.Stats.ValidShares++
							}
							s.mu.Unlock()
						}
					} else {
						// NOT a share reply. Could be auth response or eth_getWork response.
						isAuthReject := false
						isAuthReply := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							isAuthReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							isAuthReject = true
						} else if res, ok := msg["result"]; ok && res == true {
							isAuthReply = true
						} else if resultMap, ok := msg["result"].(map[string]interface{}); ok && resultMap != nil {
							isAuthReply = true // For protocols where result is an object
						}

						if isAuthReply {
							s.mu.Lock()
							if s.State == "SWITCHING_TO_FEE" {
								s.State = "FEE"
							}
							s.mu.Unlock()
						}
						
						if isAuthReject {
							s.LogBackend("[SmartRouting] Fee Pool Auth/Generic Error: %v", line)
							if s.SamePoolFeeActive {
								// We are in same-pool fee mode and got rejected.
								s.FeeAuthFailures++
								// Force reconnect
								go func() {
									s.mu.Lock()
									oldConn := s.FeeConn
									s.FeeConn = nil // Detach current connection
									s.mu.Unlock()
									if oldConn != nil {
										oldConn.Close()
									}

									// ConnectFee will automatically pick up the fallback logic since FeeAuthFailures > 0
									if s.CurrentFeeMode == FeeModeOperator {
										s.ConnectFee(s.Config.OperatorWallet, s.Config.OperatorWorker, false)
									} else {
										s.ConnectFee(s.Config.DevWallet, s.Config.DevWorker, true)
									}
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
										// Check if it's safe to rewrite
										feeTargetHashStr, _ := resArr[2].(string)
										mainDiff := parseEthProxyTargetToDiff(targetHash)
										feeDiff := parseEthProxyTargetToDiff(feeTargetHashStr)
										
										// Only rewrite if Main pool difficulty >= Fee pool difficulty
										// Otherwise the miner submits weak shares that the fee pool rejects
										if mainDiff >= feeDiff && feeDiff > 0 {
											resArr[2] = targetHash
											finalTarget = targetHash
											msg["result"] = resArr
											if modBytes, err := json.Marshal(msg); err == nil {
												line = string(modBytes)
											}
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
								if !s.IsF2PoolExploit {
									s.CurrentDiff = diffFloat
								}
								s.FeeDifficulty = diffFloat
								
								forceUpdateLocal := false
								if s.LocalDiff == 0 || diffFloat > s.LocalDiff {
									forceUpdateLocal = true
								}
								s.mu.Unlock()

								if forceUpdateLocal {
									s.mu.Lock()
									s.LocalDiff = diffFloat
									s.PendingDiff = 0
									latestFeeJob := s.LatestFeeJob
									minerConn := s.MinerConn
									s.mu.Unlock()
									
									if s.Config.EnableAsic && latestFeeJob != "" && minerConn != nil {
										// Zero-Latency Forged Job Injection for ASICs
										setDiffPkt := fmt.Sprintf(`{"id": null, "method": "mining.set_difficulty", "params": [%.0f]}`+"\n", diffFloat)
										cleanJobPkt := forceCleanJobs(latestFeeJob)
										safeFprintf(minerConn, 5*time.Second, "%s", setDiffPkt)
										safeFprintf(minerConn, 5*time.Second, "%s\n", cleanJobPkt)
										continue // Intercepted and injected manually
									}
									// Fall through for standard miners or initial connection
								} else {
									// INTERCEPT: ONLY intercept pool difficulty drops.
									continue
								}
							}
						}
					} else if method == "mining.notify" {
						s.mu.Lock()
						s.LatestFeeJob = line
						// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

						// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs
						
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
						safeFprintf(minerConn, 5*time.Second, "%s\n", line)
					} else if method, ok := msg["method"].(string); ok {
						s.mu.Lock()
						isExploit := s.IsF2PoolExploit
												
						s.mu.Unlock()
						
						if isExploit && method == "mining.set_extranonce" {
							// [F2Pool Exploit] Do NOT forward extranonce to the physical miner to prevent chip restarts.
							// The miner will continue using the Main Pool's extranonce1, which F2Pool ignores.
						} else if method == "mining.notify" || method == "mining.set_difficulty" || method == "mining.set_extranonce" || method == "eth_getWork" {
							safeFprintf(minerConn, 5*time.Second, "%s\n", line)
						}
					}
				}
			}
		}
	}(feeConn)
}

func (s *Session) EndFee() {
	s.mu.Lock()

	if s.State == "FEE" || s.State == "SWITCHING_TO_FEE" {
		s.State = "SWITCHING_TO_MAIN"
		s.TargetState = "MAIN"
	}

	s.InBandFeeActive = false
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
	
	// Write with timeout without holding the session mutex to prevent TCP block deadlocks
	// if the miner silently disconnects and the buffer fills up.
	safeFprintf(minerConn, 5*time.Second, "%s\n", string(msgBytes))
}
// SafeWrite writes data to the connection with a timeout to prevent deadlocks
func safeWrite(conn net.Conn, data []byte, timeout time.Duration) (int, error) {
	if conn == nil {
		return 0, fmt.Errorf("nil connection")
	}
	conn.SetWriteDeadline(time.Now().Add(timeout))
	n, err := conn.Write(data)
	conn.SetWriteDeadline(time.Time{})
	return n, err
}

// SafeFprintf formats according to a format specifier and writes to the connection with a timeout
func safeFprintf(conn net.Conn, timeout time.Duration, format string, a ...interface{}) (int, error) {
	if conn == nil {
		return 0, fmt.Errorf("nil connection")
	}
	return safeWrite(conn, []byte(fmt.Sprintf(format, a...)), timeout)
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
			s.mu.Unlock()

			now := time.Now()

			if now.Sub(lastShare) > 10*time.Minute && now.Sub(connAt) > 5*time.Minute {
				s.mu.Lock()
				shares := s.Stats.Shares
				s.mu.Unlock()
				if shares > 0 {
					log.Printf("[Watchdog] Miner %s timed out (no shares for 10 mins). Force closing.", s.GetMinerIdentifier())
				}
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
