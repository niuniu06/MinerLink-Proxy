package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

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
	s.SuppressNextNotify = true
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
		scanner.Split(pearlSplitFunc)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			tokenBytes := scanner.Bytes()
			if len(tokenBytes) == 0 {
				continue
			}

			if tokenBytes[0] != '{' && tokenBytes[0] != '[' {
				// Binary ACK! Forward to miner
				s.mu.Lock()
				minerConn := s.MinerConn
				s.mu.Unlock()
				if minerConn != nil {
					minerConn.Write(tokenBytes)
				}
				continue
			}

			line := string(tokenBytes)

			if s.Config.EnableDetailedLog {
				s.LogGeneral("[RAW MAIN RX] %s", strings.TrimSpace(line))
			}

			var msg map[string]interface{}
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				if s.Config.EnableAsic {
					s.mu.Lock()
					subId := s.SubscribeID
					s.mu.Unlock()
					if id, ok := msg["id"]; ok && id != nil && fmt.Sprintf("%v", id) == fmt.Sprintf("%v", subId) {
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
							s.Stats.IncInvalidShares()
						} else if res, ok := msg["result"]; ok && res == true || transitionMasked {
							if !transitionMasked {
								s.Stats.IncValidShares()
							}
							s.mu.Lock()
							if isFee && pending.FeeMode == FeeModeOperator {
								s.Stats.IncFeeShares()
							}
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							if isFee {
								s.RingBuffer.AddShare(s.CurrentDiff, true, pending.FeeMode == FeeModeDev)
								if s.Server != nil {
									s.Server.RingBuffer.AddShare(s.CurrentDiff, true, pending.FeeMode == FeeModeDev)
								}
							} else {
								s.RingBuffer.AddShare(s.CurrentDiff, false, false)
								if s.Server != nil {
									s.Server.RingBuffer.AddShare(s.CurrentDiff, false, false)
								}
							}
							s.mu.Unlock()
						}

						if isFee {
							if isReject {
								if s.Config.EnableDetailedLog {
									s.LogError("[FEE] share rejected! Pool Response: %s | Original Request: %s", strings.TrimSpace(line), origReq)
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
								s.mu.Lock()
								cMode := s.CurrentFeeMode
								s.mu.Unlock()
								s.addJob(powHash, cMode)
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
								suppress := s.SuppressNextNotify
								s.mu.Unlock()

								if inBandActive {
									// INTERCEPT: Do not forward unexpected difficulty resets from the pool
									// during In-Band fee routing, as it causes ASIC hashrate drops/restarts.
									s.LogBackend("[SmartRouting] Intercepted pool difficulty drop (%.0f) during In-Band Fee. Miner kept at %.0f", diffFloat, s.MainDifficulty)
									continue
								}
								if suppress {
									s.mu.Lock()
									s.CurrentDiff = diffFloat
									s.MainDifficulty = diffFloat
									s.RemoteDiff = diffFloat
									s.mu.Unlock()
									continue // Intercept initial diff from auto-reconnect
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
						suppress := s.SuppressNextNotify
						if suppress {
							s.SuppressNextNotify = false
						}
						s.mu.Unlock()

						// Rate limiter for clean_jobs: false
						isCleanJobs := false
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
							if cj, ok := params[8].(bool); ok {
								isCleanJobs = cj
							}
						} else {
							// If we can't parse it, assume it's clean to be safe
							isCleanJobs = true
						}

						if !isCleanJobs {
							now := time.Now()
							s.mu.Lock()
							elapsed := now.Sub(s.LastNotifyTime)
							s.mu.Unlock()
							if elapsed < 5*time.Second {
								s.LogGeneral("[Anti-Crash] Dropped high-frequency clean_jobs:false notify (interval: %v)", elapsed)
								continue
							}
						}

						s.mu.Lock()
						s.LastNotifyTime = time.Now()
						s.mu.Unlock()

						// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

						// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if jobID, ok := params[0].(string); ok {
								s.addJob(jobID, FeeModeNone)
							}
						} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
							if jobID, ok := paramsMap["job_id"].(string); ok {
								s.addJob(jobID, FeeModeNone)
							}
						}

						if suppress {
							continue // Intercept initial notify from auto-reconnect
						}
					}
				}
			}

			s.mu.Lock()
			state := s.State
			minerConn := s.MinerConn
			s.mu.Unlock()

			shouldForward := (state == "MAIN" || state == "SWITCHING_TO_MAIN")
			

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
					idStr := fmt.Sprintf("%v", id)
					if isLoginPacket {
						if s.ForwardedResponseIDs == nil {
							s.ForwardedResponseIDs = make(map[string]bool)
						}
						if s.ForwardedResponseIDs[idStr] {
							shouldForward = false
						} else {
							s.ForwardedResponseIDs[idStr] = true
						}
					} else {
						// [CRITICAL BUGFIX] Intercept proxy-injected ghost IDs
						// When reconnecting to the pool (or switching fee routing), the proxy uses high IDs like 99998/99999
						// to avoid colliding with the miner's original packets. The pool replies to these ghost IDs.
						// If we forward these ghost replies to strict miners (S19/S21/ETC), they will immediately crash/disconnect.
						if idStr == "99998" || idStr == "99999" || idStr == "999999" {
							shouldForward = false
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

