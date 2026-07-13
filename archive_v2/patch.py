import io

with io.open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    content = f.read()

target = '''					}
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
								s.mu.Unlock()
								GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

								// Vardiff Bugfix: If Vardiff is active, DO NOT forward mining.set_difficulty immediately!'''

replacement = '''					}
				} else if method, ok := msg["method"].(string); ok {
					if method == "mining.set_version_mask" {
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 0 {
							if mask, ok := params[0].(string); ok {
								s.mu.Lock()
								isDuplicate := (s.MainVersionMask == mask)
								s.MainVersionMask = mask
								s.mu.Unlock()
								if isDuplicate {
									// [Bugfix] Filter out redundant set_version_mask
									continue
								}
							}
						}
					} else if method == "mining.set_difficulty" {
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
								isDuplicate := (s.CurrentDiff == diffFloat)
								s.CurrentDiff = diffFloat
								s.MainDifficulty = diffFloat
								s.RemoteDiff = diffFloat
								s.mu.Unlock()
								GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

								if isDuplicate && !enableVardiff {
									// [Bugfix] Filter out redundant set_difficulty that causes Antminer to drop connection
									continue
								}

								// Vardiff Bugfix: If Vardiff is active, DO NOT forward mining.set_difficulty immediately!'''

content = content.replace(target, replacement)

with io.open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(content)
