package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"proxy-core/internal/models"
)

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

type Session struct {
	ID             string
	MinerConn      net.Conn
	MainConn       net.Conn
	FeeConn        net.Conn
	Config         *models.ProxyConfig
	
	MinerWallet    string
	MinerWorker    string
	
	State          string // "MAIN", "SWITCHING_TO_FEE", "FEE", "SWITCHING_TO_MAIN"
	TargetState    string
	CurrentFeeMode FeeMode
	
	Stats          SessionStats
	
	ShareHistory   []ShareEvent
	CurrentDiff    float64
	LastHashUpdate time.Time
	DisplayHash    float64
	
	// Locks and sync
	mu             sync.Mutex
	quit           chan struct{}
	
	// Protocols
	loginPackets   []map[string]interface{}
	pendingShares  sync.Map
	cycleOffset    int
	JobSourceMap   sync.Map // JobID (string) -> bool (isMain)

	// ASIC Extranonce Support
	SubscribeID    interface{}
	MainExtranonce *ExtranonceData
	FeeExtranonce  *ExtranonceData

	// Zero-Latency Switching
	LatestMainJob string
	LatestFeeJob  string
	IsPreWarmed   bool
}

func NewSession(conn net.Conn, cfg *models.ProxyConfig) *Session {
	return &Session{
		ID:             fmt.Sprintf("%d", time.Now().UnixNano()),
		MinerConn:      conn,
		Config:         cfg,
		State:          "MAIN",
		TargetState:    "MAIN",
		CurrentFeeMode: FeeModeNone,
		Stats:          SessionStats{ConnectedAt: time.Now()},
		quit:           make(chan struct{}),
		loginPackets:   make([]map[string]interface{}, 0),
		cycleOffset:    -1,
		ShareHistory:   make([]ShareEvent, 0),
		CurrentDiff:    1.0,
		LastHashUpdate: time.Now(),
	}
}

func (s *Session) Start() {
	log.Printf("[Miner %s] Connected", s.ID)
	// Connect to main pool
	var err error
	s.MainConn, err = net.Dial("tcp", s.Config.PoolAddress)
	if err != nil {
		log.Printf("[Miner %s] Failed to connect to main pool: %v", s.ID, err)
		s.Close()
		return
	}
	
	go s.timerLoop()
	go s.readMainLoop()
	s.readMinerLoop()
}

func (s *Session) Close() {
	log.Printf("[Miner %s] Session Close called", s.ID)
	s.mu.Lock()
	defer s.mu.Unlock()
	
	select {
	case <-s.quit:
		return
	default:
		close(s.quit)
	}

	if s.MinerConn != nil { s.MinerConn.Close() }
	if s.MainConn != nil { s.MainConn.Close() }
	if s.FeeConn != nil { s.FeeConn.Close() }
}

func (s *Session) FormatHashrate() string {
	// If HashrateMultiplier is set, fake it
	multiplier := s.Config.HashrateMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}
	
	// Calculate rolling window hashrate (10 minutes)
	now := time.Now()
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
		
		// 4.294967296 is 2^32 / 10^9 (GH/s base formula for stratum difficulty)
		s.DisplayHash = (diffSum * 4.294967296) / window * 1000 // Convert to MH/s as base
		s.LastHashUpdate = now
		s.mu.Unlock()
	}
	
	hs := s.DisplayHash * multiplier
	
	unit := s.Config.HashrateUnit
	if unit == "" {
		if hs > 1000 {
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
	buf := ScannerBufferPool.Get().([]byte)
	buf = buf[:0]
	defer ScannerBufferPool.Put(buf)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		
		var method string
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			method, _ = msg["method"].(string)
			if method == "mining.subscribe" || method == "eth_submitLogin" || method == "mining.authorize" || method == "login" {
				// Deep copy msg to store in loginPackets so it isn't mutated by MainFixedDifficulty
				var pktCopy map[string]interface{}
				pktBytes, _ := json.Marshal(msg)
				json.Unmarshal(pktBytes, &pktCopy)
				s.loginPackets = append(s.loginPackets, pktCopy)
				if method == "mining.subscribe" {
					s.mu.Lock()
					s.SubscribeID = msg["id"]
					s.mu.Unlock()
				}
				if method == "mining.authorize" || method == "eth_submitLogin" || method == "login" {
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if pStr, ok := params[0].(string); ok {
							parts := strings.Split(pStr, ".")
							s.MinerWallet = parts[0]
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
					} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
						if w, ok := paramsMap["wallet"].(string); ok {
							s.MinerWallet = w
						}
						if w, ok := paramsMap["worker"].(string); ok {
							s.MinerWorker = w
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
				if id, ok := msg["id"]; ok {
					s.pendingShares.Store(id, true)
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
				if isMainRaw, ok := s.JobSourceMap.Load(submitJobID); ok {
					isMainRoute = isMainRaw.(bool)
				}
			}

			if isMainRoute {
				if mainConn != nil {
					fmt.Fprintf(mainConn, "%s\n", line)
				}
			} else {
				if feeConn != nil {
					fmt.Fprintf(feeConn, "%s\n", line)
				} else {
					// Fee pool disconnected, rescue via fake accept
					if id, ok := msg["id"]; ok {
						s.pendingShares.Delete(id)
						fakeReply := fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
						s.MinerConn.Write([]byte(fakeReply))
						log.Printf("[Miner %s] Perfect Routing: Fake accepted late fee share (%s) because fee connection is closed", s.ID, submitJobID)
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
	buf := ScannerBufferPool.Get().([]byte)
	buf = buf[:0]
	defer ScannerBufferPool.Put(buf)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		
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
				if _, exists := s.pendingShares.LoadAndDelete(id); exists {
					if errObj, ok := msg["error"]; ok && errObj != nil {
						s.Stats.InvalidShares++
					} else if res, ok := msg["result"]; ok && res == false {
						s.Stats.InvalidShares++
					} else {
						s.Stats.ValidShares++
						s.mu.Lock()
						s.ShareHistory = append(s.ShareHistory, ShareEvent{
							Timestamp: time.Now(),
							Diff:      s.CurrentDiff,
						})
						s.mu.Unlock()
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
					s.LatestMainJob = line
					s.mu.Unlock()
					if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
						if jobID, ok := params[0].(string); ok {
							s.JobSourceMap.Store(jobID, true) // true = Main
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
					// Ensure at least 3 minutes (180s) grace period before fee triggers
					maxR := nonFeeLength - 180
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
			if targetMode != s.CurrentFeeMode {
				s.CurrentFeeMode = targetMode
				s.IsPreWarmed = false // Reset pre-warm flag

				if targetMode == FeeModeNone {
					s.TargetState = "MAIN"
					s.State = "SWITCHING_TO_MAIN"
					go s.EndFee()
					if s.Config.EnableAsic {
						en := s.MainExtranonce
						s.sendExtranonce(en)
					}
					// Zero-latency job injection
					cachedJob := s.LatestMainJob
					minerConn := s.MinerConn
					if cachedJob != "" && minerConn != nil {
						fmt.Fprintf(minerConn, "%s\n", cachedJob)
					}
				} else if targetMode == FeeModeDev || targetMode == FeeModeOperator {
					s.TargetState = "FEE"
					s.State = "SWITCHING_TO_FEE"
					if s.Config.EnableAsic {
						en := s.FeeExtranonce
						s.sendExtranonce(en)
					}
					// Zero-latency job injection
					cachedJob := s.LatestFeeJob
					minerConn := s.MinerConn
					if cachedJob != "" && minerConn != nil {
						fmt.Fprintf(minerConn, "%s\n", cachedJob)
					}
				}
			}
			s.mu.Unlock()
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

	log.Printf("[Miner %s] Connecting to Fee Pool for %s", s.ID, wallet)
	
	// Create fee connection
	host := s.Config.PoolAddress
	feeConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("[Miner %s] Fee connection failed: %v", s.ID, err)
		s.EndFee()
		return
	}

	s.mu.Lock()
	s.FeeConn = feeConn
	s.mu.Unlock()

	// Replay login packets
	for _, pkt := range s.loginPackets {
		// Deep copy to not mutate original
		pktBytes, _ := json.Marshal(pkt)
		var mod map[string]interface{}
		json.Unmarshal(pktBytes, &mod)

		method, _ := mod["method"].(string)
		if method == "mining.authorize" || method == "eth_submitLogin" || method == "login" {
			if params, ok := mod["params"].([]interface{}); ok && len(params) > 0 {
				if originalUser, ok := params[0].(string); ok {
					if strings.Contains(originalUser, ".") {
						mod["params"].([]interface{})[0] = fmt.Sprintf("%s.%s", wallet, worker)
					} else {
						mod["params"].([]interface{})[0] = wallet
						// Forcefully inject worker as the second parameter
						if len(params) > 1 {
							mod["params"].([]interface{})[1] = worker
						} else {
							mod["params"] = append(mod["params"].([]interface{}), worker)
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
			
			// Also aggressively inject at the root level for some miner variants
			mod["worker"] = worker

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

	// Read loop
	go func() {
		defer s.EndFee()
		scanner := bufio.NewScanner(feeConn)
		buf := ScannerBufferPool.Get().([]byte)
		buf = buf[:0]
		defer ScannerBufferPool.Put(buf)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			
			var msg map[string]interface{}
			var isShareReply bool
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
									s.mu.Unlock()
									s.sendExtranonce(en)
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
									s.mu.Unlock()
									s.sendExtranonce(en)
								}
							}
						}
					}
				}

				if id, ok := msg["id"]; ok && id != nil {
					if _, exists := s.pendingShares.LoadAndDelete(id); exists {
						isShareReply = true
						if errObj, ok := msg["error"]; ok && errObj != nil {
							s.Stats.InvalidShares++
						} else if res, ok := msg["result"]; ok && res == false {
							s.Stats.InvalidShares++
						} else {
							s.Stats.FeeShares++
							s.Stats.ValidShares++
							s.mu.Lock()
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.mu.Unlock()
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
								s.JobSourceMap.Store(jobID, false) // false = Fee
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
					// Only forward specific methods or share replies, NOT login replies
					if isShareReply {
						fmt.Fprintf(minerConn, "%s\n", line)
					} else if method, ok := msg["method"].(string); ok {
						if method == "mining.notify" || method == "mining.set_difficulty" || method == "mining.set_extranonce" || method == "eth_getWork" || method == "eth_getWork" {
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
	if extranonce == nil || s.MinerConn == nil { return }
	msg := map[string]interface{}{
		"id":     nil,
		"method": "mining.set_extranonce",
		"params": []interface{}{extranonce.En1, extranonce.En2Size},
	}
	msgBytes, _ := json.Marshal(msg)
	s.mu.Lock()
	fmt.Fprintf(s.MinerConn, "%s\n", string(msgBytes))
	s.mu.Unlock()
}

