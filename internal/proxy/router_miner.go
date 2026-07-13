package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"proxy-core/internal/db"
)

func (s *Session) readMinerLoop() {
	defer s.Close()
	scanner := bufio.NewScanner(s.MinerConn)
	scanner.Split(pearlSplitFunc)
	bufPtr := ScannerBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]
	defer ScannerBufferPool.Put(bufPtr)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		tokenBytes := scanner.Bytes()
		if len(tokenBytes) == 0 {
			continue
		}

		if tokenBytes[0] != '{' && tokenBytes[0] != '[' {
			// Binary byte! Optimistic forward to current active pool
			s.mu.Lock()
			targetConn := s.MainConn
			if s.State == "FEE" && s.FeeConn != nil {
				targetConn = s.FeeConn
			}
			isFee := s.State == "FEE"
			isDev := s.CurrentFeeMode == FeeModeDev
			diff := s.CurrentDiff

			s.BinaryShareBytes += uint64(len(tokenBytes))
			addedShare := false
			if s.BinaryShareBytes%6 == 0 {
				addedShare = true
				s.PhysicalShares++
				s.Stats.IncShares()
				s.LastShareTime = time.Now()
				s.ShareHistory = append(s.ShareHistory, ShareEvent{
					Timestamp: time.Now(),
					Diff:      diff,
				})
			}
			s.mu.Unlock()

			if targetConn != nil {
				targetConn.Write(tokenBytes)
			}

			if addedShare && s.Server != nil {
				if diff <= 0 {
					diff = 1.0
				}
				s.Server.RingBuffer.AddShare(diff, isFee, isDev)
			}
			continue
		}

		line := string(tokenBytes)

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
					isEthFamily := (expectedCoin == "ETC" || expectedCoin == "ETHW")
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

						// Fallback: Some miners pack the worker into the wallet field (e.g. "wallet.worker")
						// even when using map-based params, without sending a separate "worker" field.
						if s.MinerWorker == "" && strings.Contains(s.MinerWallet, ".") {
							parts := strings.SplitN(s.MinerWallet, ".", 2)
							s.MinerWallet = parts[0]
							s.MinerWorker = parts[1]
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
							s.Stats.SetConnectedAt(time.Now())
							s.ForwardedResponseIDs = make(map[string]bool)
							if s.Stats.ValidShares() > 0 {
								// Keep the offline state logic intact
							}
							s.mu.Unlock()
							s.LogGeneral("Miner session restored from offline state, inherited %d valid shares", oldStats.ValidShares())
							cName := ""
							if s.Config != nil {
								cName = s.Config.CoinName
							}
							db.RecordEvent(s.GetMinerIP(), s.MinerWorker, cName, s.MinerWallet, "ONLINE", "Miner reconnected from offline state")
						} else {
							s.LogGeneral("Miner authorized: %s", s.MinerWorker)
							cName := ""
							if s.Config != nil {
								cName = s.Config.CoinName
							}
							db.RecordEvent(s.GetMinerIP(), s.MinerWorker, cName, s.MinerWallet, "ONLINE", "Miner successfully authorized")
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
				s.Stats.IncShares()
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
				} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
					if jobIDStr, ok := paramsMap["job_id"].(string); ok {
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
			var shareFeeMode FeeMode = FeeModeNone

			if submitJobID != "" {
				if mode, exists := s.checkJobFeeMode(submitJobID); exists {
					isMainRoute = (mode == FeeModeNone)
					shareFeeMode = mode
				}
			}

			s.mu.Lock()
			isExploit := s.IsF2PoolExploit
			inBandFeeActive := s.InBandFeeActive
			s.mu.Unlock()

			// [SmartRouting]: If the job is unknown, fallback to heuristics.
			// We force intercept if we are in FEE state under Exploit or InBand modes.
			if submitJobID == "" {
				if (isExploit || inBandFeeActive) && (state == "FEE" || state == "SWITCHING_TO_FEE") {
					isMainRoute = false
					s.mu.Lock()
					shareFeeMode = s.CurrentFeeMode
					s.mu.Unlock()
				}
			}

			// Pure Smoothed Ghost Routing overrides MainRoute removed (PRL now uses standard time-based switching)

			if isMainRoute {
				if mainConn != nil {
					if id, ok := msg["id"]; ok {
						s.pendingShares.Store(id, ShareContext{Req: strings.TrimSpace(line), IsFee: false, FeeMode: FeeModeNone})
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
						s.pendingShares.Store(id, ShareContext{Req: strings.TrimSpace(line), IsFee: true, FeeMode: shareFeeMode})
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
