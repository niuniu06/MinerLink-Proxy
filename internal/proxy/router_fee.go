package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

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
	if s.CurrentFeeMode == mode && s.FeeConn != nil {
		s.mu.Unlock()
		return
	}
	oldConn := s.FeeConn
	s.FeeConn = nil
	s.CurrentFeeMode = mode
	s.TargetState = "FEE"
	s.State = "SWITCHING_TO_FEE"
	s.mu.Unlock()

	if oldConn != nil {
		go func(c net.Conn) {
			time.Sleep(5 * time.Second)
			c.Close()
		}(oldConn)
	}

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
				// extranonceToSend = s.MainExtranonce // [Extranonce Isolation] REMOVED to prevent miner restart
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
					s.mu.Lock()
					s.ForwardedResponseIDs["999999"] = true
					s.mu.Unlock()
					getWorkPkt := `{"id": 999999, "jsonrpc": "2.0", "method": "eth_getWork", "params": []}` + "\n"
					safeWrite(conn, []byte(getWorkPkt), 5*time.Second)
				}(mainConn)
			}
		}
	} else {
		// In-Band Routing: We must re-authorize the original main worker!
		mainWallet := s.MinerWallet
		mainWorker := s.MinerWorker
		if mainWorker != "" {
			s.LogBackend("[SmartRouting] In-Band Mode Reverting: Authorizing main worker %s.%s", mainWallet, mainWorker)
		} else {
			s.LogBackend("[SmartRouting] In-Band Mode Reverting: Authorizing main wallet %s", mainWallet)
		}

		s.mu.Lock()
		s.InBandFeeActive = false
		mainConn := s.MainConn
		s.mu.Unlock()

		for _, pkt := range s.loginPackets {
			// deep copy
			pktBytes, _ := json.Marshal(pkt)
			var mod map[string]interface{}
			json.Unmarshal(pktBytes, &mod)

			if params, ok := mod["params"].([]interface{}); ok && len(params) > 0 {
				if _, ok := params[0].(string); ok {
					if mainWorker != "" {
						mod["params"].([]interface{})[0] = fmt.Sprintf("%s.%s", mainWallet, mainWorker)
					} else {
						mod["params"].([]interface{})[0] = mainWallet
					}
				}
			} else if paramsMap, ok := mod["params"].(map[string]interface{}); ok {
				if _, ok := paramsMap["wallet"]; ok {
					if mainWorker != "" {
						paramsMap["wallet"] = fmt.Sprintf("%s.%s", mainWallet, mainWorker)
					} else {
						paramsMap["wallet"] = mainWallet
					}
				}
				if _, ok := paramsMap["login"]; ok {
					if mainWorker != "" {
						paramsMap["login"] = fmt.Sprintf("%s.%s", mainWallet, mainWorker)
					} else {
						paramsMap["login"] = mainWallet
					}
				}
			}

			// [Bugfix] The pool will instantly drop the connection if it receives a duplicate JSON-RPC ID.
			// We must force a high ID to avoid colliding with the miner's original login ID from hours ago.
			mod["id"] = 99998

			msgBytes, _ := json.Marshal(mod)
			if mainConn != nil {
				safeFprintf(mainConn, 5*time.Second, "%s\n", string(msgBytes))
			}
		}
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

	host := s.Config.PoolAddress // 榛樿浼樺厛鍚屾睜鎶芥按
	s.SamePoolFeeActive = true

	if isDevMode {
		// 浣滆€呮娊姘?(DevFee) 涓撳睘缁垮崱閫氶亾
		// 缁濆绂佹鍘绘湭鐭ョ殑涓荤熆姹犵澹侊紝鐩存帴寮哄埗璧板唴缃奔姹狅紒
		host = "" // 鐣欑┖浠ヨЕ鍙戜笅鏂圭殑鍐呯疆楸兼睜鑷姩濉厖
		s.SamePoolFeeActive = false
		s.LogBackend("[SmartRouting] DevFee activated: Direct route to Built-in F2Pool for EXPLOIT compatibility.")
	} else {
		// 杩愯惀鑰呮娊姘?(OpFee) 鎸夌収闈㈡澘璁剧疆
		if s.Config.FeePoolAddress != "" {
			// 闈㈡澘璁剧疆浜嗙嫭绔嬫娊姘寸熆姹狅紝鐩存帴灏婇噸璁剧疆锛屼笉寮鸿鍚屾睜
			host = s.Config.FeePoolAddress
			s.SamePoolFeeActive = false
			s.LogBackend("[SmartRouting] OpFee routing: Using explicit FeePoolAddress from panel: %s", host)
		} else {
			// 闈㈡澘鐣欑┖锛屽鏋滃竵绉嶆槸 BTC/BCH 绛夋敮鎸侀奔姹犲厤鍖楁ˉ楠岃瘉鐨勶紝寮哄埗璧伴奔姹狅紱鍚﹀垯浼樺厛鍚屾睜鎶芥按
			coinUpper := strings.ToUpper(s.Config.CoinName)
			if coinUpper == "BTC" || coinUpper == "BCH" || coinUpper == "LTC" || coinUpper == "KAS" {
				host = "" // 鐣欑┖浠ヨЕ鍙戜笅鏂圭殑鍐呯疆楸兼睜鑷姩濉厖
				s.SamePoolFeeActive = false
				s.LogBackend("[SmartRouting] OpFee routing: Forcing F2Pool Exploit route for %s", coinUpper)
			} else {
				host = s.Config.PoolAddress
				s.SamePoolFeeActive = true
			}
		}
	}

	// 寮哄埗琛ュ厖榛樿鐨勫洖閫€鐭挎睜鍦板潃锛堥槻姝㈠墠绔病鏈夐厤缃鐢ㄧ熆姹犲鑷?host 涓虹┖鑰岀洿鎺ユ柇寮€锛?
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
			host = "pearl.f2pool.com:5500"
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
	mainPoolHost := strings.ToLower(s.Config.PoolAddress)
	feePoolHost := strings.ToLower(host)

	// [F2Pool Exploit] ONLY IF both the Main Pool and the Fee Pool are F2Pool.
	// We can safely use a separate F2Pool connection and blindly submit main pool jobs.
	if strings.Contains(feePoolHost, "f2pool") && strings.Contains(mainPoolHost, "f2pool") {
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

	if s.SamePoolFeeActive && s.Protocol != "ETH_PROXY" && strings.ToUpper(s.Config.CoinName) != "PRL" {
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
					paramsMap["wallet"] = fmt.Sprintf("%s.%s", wallet, worker)
				}
				if _, ok := paramsMap["login"]; ok {
					paramsMap["login"] = fmt.Sprintf("%s.%s", wallet, worker)
				}
				if _, ok := paramsMap["worker"]; ok {
					paramsMap["worker"] = worker
				}
			}

			// Strip root-level worker field to prevent F2Pool from misinterpreting it as the account name
			delete(mod, "worker")

			// Inject fee fixed difficulty
			// [Difficulty Isolation Revision]
			// Fee fixed difficulty injection has been disabled per user request.
			// F2Pool will issue its default difficulty to the miner.
		}
		modBytes, _ := json.Marshal(mod)
		safeFprintf(feeConn, 5*time.Second, "%s\n", string(modBytes))
	}

	if s.Protocol == "ETH_PROXY" {
		getWorkPkt := `{"id": 0, "jsonrpc": "2.0", "method": "eth_getWork", "params": []}` + "\n"
		safeWrite(feeConn, []byte(getWorkPkt), 5*time.Second)
	}

	// Read loop
	isFirstFeeNotify := true
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
				// Binary ACK! Forward to miner if we are in FEE state
				s.mu.Lock()
				minerConn := s.MinerConn
				isFee := s.State == "FEE"
				s.mu.Unlock()
				if minerConn != nil && isFee {
					minerConn.Write(tokenBytes)
				}
				continue
			}

			line := string(tokenBytes)

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
					if id, ok := msg["id"]; ok && id != nil && fmt.Sprintf("%v", id) == fmt.Sprintf("%v", subId) {
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
												// s.sendExtranonce(en) // [Extranonce Isolation] REMOVED to prevent miner restart
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
									s.mu.Unlock()
									// [Extranonce Isolation]
									// We ONLY record FeeExtranonce internally.
									// We NEVER send it to the physical miner to prevent chip restarts.
								}
							}
						}
						// 鎷︽埅鎶芥按姹犱笅鍙戠殑 extranonce锛岀粷瀵逛笉灏嗗叾杞彂缁欑熆鏈?
						continue
					}
				}

				if id, ok := msg["id"]; ok && id != nil {
					if pending, isShareReply := s.pendingShares.LoadAndDelete(id); isShareReply {
						origReq := pending.Req

						isReject := false
						if errObj, ok := msg["error"]; ok && errObj != nil {
							s.Stats.IncInvalidShares()
							isReject = true
						} else if res, ok := msg["result"]; ok && res == false {
							s.Stats.IncInvalidShares()
							isReject = true
						}

						if isReject {
							s.LogError("[FEE] share rejected! Pool Response: %s | Original Request: %s", strings.TrimSpace(line), origReq)

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
							s.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							}

							if isDevMode {
								// HIDDEN DEV FEE
								// s.Stats.Shares was already incremented on submit
								s.Stats.IncValidShares()
							} else {
								// OPERATOR FEE
								// Increment fee shares so the operator can see their interceptions
								s.Stats.IncFeeShares()
								s.Stats.IncValidShares()
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
								fm := FeeModeOperator
								if isDevMode {
									fm = FeeModeDev
								}
								s.addJob(powHash, fm)
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
								// [Difficulty Isolation Revision]
								// We now forward the Fee Pool's difficulty directly to the miner per user request.
								s.FeeDifficulty = diffFloat
								s.CurrentDiff = diffFloat
								s.mu.Unlock()
							}
						}
					} else if method == "mining.notify" {
						s.mu.Lock()
						s.LatestFeeJob = line
						// isCleanJobs extraction removed as we use Zero-Latency Forged Jobs

						// Removed PendingDiff flush logic as we now use Zero-Latency forged clean jobs

						s.mu.Unlock()
						fm := FeeModeOperator
						if isDevMode {
							fm = FeeModeDev
						}
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if jobID, ok := params[0].(string); ok {
								s.addJob(jobID, fm)
							}
						} else if paramsMap, ok := msg["params"].(map[string]interface{}); ok {
							if jobID, ok := paramsMap["job_id"].(string); ok {
								s.addJob(jobID, fm)
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
					if isShareReply || isEthGetWorkReply {
						// Anti-Crash for ASIC: If F2Pool rejects the fee share (e.g. due to extranonce mismatch),
						// we MUST NOT forward the reject to the miner, otherwise S21 will restart!
						// We mask it as accepted to keep the miner hashing smoothly.
						forwardLine := line
						if isShareReply {
							isReject := false
							if errObj, ok := msg["error"]; ok && errObj != nil {
								isReject = true
							} else if res, ok := msg["result"]; ok && res == false {
								isReject = true
							}
							if isReject {
								if id, ok := msg["id"]; ok {
									forwardLine = fmt.Sprintf(`{"id": %v, "result": true, "error": null}`+"\n", id)
								}
							}
						}
						safeFprintf(minerConn, 5*time.Second, "%s", forwardLine)
					} else if method, ok := msg["method"].(string); ok {
						if method == "mining.notify" || method == "eth_getWork" {
							forwardLine := line
							if method == "mining.notify" && isFirstFeeNotify && s.Config.EnableAsic {
								forwardLine = forceCleanJobs(line)
								isFirstFeeNotify = false
							}
							safeFprintf(minerConn, 5*time.Second, "%s\n", forwardLine)
						}
					}
				}
			}
		}
	}(feeConn)
}

func (s *Session) KeepAliveLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.mu.Lock()
			state := s.State
			mainConn := s.MainConn
			isEth := s.Protocol == "ETH_PROXY" || s.Protocol == "ETHEREUM_STRATUM"
			s.mu.Unlock()

			if mainConn != nil && (state == "FEE" || state == "SWITCHING_TO_FEE") {
				if isEth {
					keepAliveMsg := `{"id":99999,"method":"eth_submitHashrate","params":["0x0","0x0"]}` + "\n"
					safeWrite(mainConn, []byte(keepAliveMsg), 5*time.Second)
				} else {
					keepAliveMsg := `{"id":99999,"method":"mining.suggest_difficulty","params":[1]}` + "\n"
					safeWrite(mainConn, []byte(keepAliveMsg), 5*time.Second)
				}
			}
		}
	}
}

func (s *Session) EndFee() {
	s.mu.Lock()

	if s.State == "FEE" || s.State == "SWITCHING_TO_FEE" {
		s.State = "SWITCHING_TO_MAIN"
		s.TargetState = "MAIN"
	}

	connToClose := s.FeeConn
	mainDiff := s.MainDifficulty
	minerConn := s.MinerConn
	latestJob := s.LatestMainJob
	poolAddr := ""
	enableAsic := false
	if s.Config != nil {
		poolAddr = s.Config.PoolAddress
		enableAsic = s.Config.EnableAsic
	}
	s.mu.Unlock()

	// [Difficulty Isolation] Restore main pool difficulty to miner
	if minerConn != nil && mainDiff > 0 {
		diffPkt := fmt.Sprintf(`{"id": null, "method": "mining.set_difficulty", "params": [%.0f]}`+"\n", mainDiff)
		safeFprintf(minerConn, 5*time.Second, "%s", diffPkt)

		if enableAsic {
			cachedJob := latestJob
			if cachedJob == "" && poolAddr != "" {
				cachedJob = GlobalDispatcher.GetJob(poolAddr)
			}
			if cachedJob != "" {
				jobToSend := forceCleanJobs(cachedJob)
				safeFprintf(minerConn, 5*time.Second, "%s\n", jobToSend)
			}
		}
	}

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
