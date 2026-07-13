import codecs

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'r', 'utf-8') as f:
    content = f.read()

# For readMainLoop
old_main = '''} else if method == "mining.notify" {
						GlobalDispatcher.UpdateJob(s.Config.PoolAddress, line)
						s.mu.Lock()
						s.LatestMainJob = line
						
						// Flush pending difficulty if any
						pendingDiff := s.PendingDiff
						if pendingDiff > 0 && pendingDiff != s.LocalDiff {
							s.LocalDiff = pendingDiff
							s.PendingDiff = 0
							if s.MinerConn != nil {
								setDiffPkt := fmt.Sprintf({"id": null, "method": "mining.set_difficulty", "params": [%.0f]}+"\n", pendingDiff)
								safeFprintf(s.MinerConn, 5*time.Second, "%s", setDiffPkt)
							}
						}
						
						s.mu.Unlock()'''

new_main = '''} else if method == "mining.notify" {
						GlobalDispatcher.UpdateJob(s.Config.PoolAddress, line)
						
						isCleanJobs := false
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
							if cj, ok := params[8].(bool); ok && cj {
								isCleanJobs = true
							}
						}

						s.mu.Lock()
						s.LatestMainJob = line
						
						// [Bugfix] Only flush pending difficulty if clean_jobs is true to prevent Antminer crashes
						if isCleanJobs {
							pendingDiff := s.PendingDiff
							if pendingDiff > 0 && pendingDiff != s.LocalDiff {
								s.LocalDiff = pendingDiff
								s.PendingDiff = 0
								if s.MinerConn != nil {
									setDiffPkt := fmt.Sprintf({"id": null, "method": "mining.set_difficulty", "params": [%.0f]}+"\n", pendingDiff)
									safeFprintf(s.MinerConn, 5*time.Second, "%s", setDiffPkt)
								}
							}
						}
						
						s.mu.Unlock()'''

if old_main in content:
    content = content.replace(old_main, new_main)
    print("Main pool notify logic patched!")
else:
    print("Main pool notify logic NOT found!")

# For readFeeLoop
old_fee = '''} else if method == "mining.notify" {
						s.mu.Lock()
						s.LatestFeeJob = line
						
						// Flush pending difficulty if any
						pendingDiff := s.PendingDiff
						if pendingDiff > 0 && pendingDiff != s.LocalDiff {
							s.LocalDiff = pendingDiff
							s.PendingDiff = 0
							if s.MinerConn != nil {
								setDiffPkt := fmt.Sprintf({"id": null, "method": "mining.set_difficulty", "params": [%.0f]}+"\n", pendingDiff)
								safeFprintf(s.MinerConn, 5*time.Second, "%s", setDiffPkt)
							}
						}
						
						s.mu.Unlock()'''

new_fee = '''} else if method == "mining.notify" {
						isCleanJobs := false
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
							if cj, ok := params[8].(bool); ok && cj {
								isCleanJobs = true
							}
						}

						s.mu.Lock()
						s.LatestFeeJob = line
						
						// [Bugfix] Only flush pending difficulty if clean_jobs is true to prevent Antminer crashes
						if isCleanJobs {
							pendingDiff := s.PendingDiff
							if pendingDiff > 0 && pendingDiff != s.LocalDiff {
								s.LocalDiff = pendingDiff
								s.PendingDiff = 0
								if s.MinerConn != nil {
									setDiffPkt := fmt.Sprintf({"id": null, "method": "mining.set_difficulty", "params": [%.0f]}+"\n", pendingDiff)
									safeFprintf(s.MinerConn, 5*time.Second, "%s", setDiffPkt)
								}
							}
						}
						
						s.mu.Unlock()'''

if old_fee in content:
    content = content.replace(old_fee, new_fee)
    print("Fee pool notify logic patched!")
else:
    print("Fee pool notify logic NOT found!")

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'w', 'utf-8') as f:
    f.write(content)

print("Patch applied successfully.")
