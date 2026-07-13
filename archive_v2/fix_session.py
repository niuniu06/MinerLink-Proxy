import re

with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

# 1. Fix StopFeeMining
# Remove the early return for isExploit
code = re.sub(r'(\t+isExploit := s\.IsF2PoolExploit\n\t+s\.mu\.Unlock\(\)\n\n\t+if isExploit \{\n[\s\S]*?\t+return\n\t+\}\n)',
              r'\t+isExploit := s.IsF2PoolExploit\n\t+s.mu.Unlock()\n', code)

# Fix the extranonce condition to exclude isExploit, and set difficultyToSend to LocalDiff
old_stop_block = '''	if !s.InBandFeeActive {
		if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
			if s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size || s.FeeExtranonce.En1 != s.MainExtranonce.En1 {
				extranonceToSend = s.MainExtranonce
				s.LastExtranonceCmdTime = time.Now()
			}
			if s.MainDifficulty > 0 && s.MainDifficulty != s.CurrentDiff {
				difficultyToSend = s.MainDifficulty
				s.CurrentDiff = s.MainDifficulty
			}
		}'''

new_stop_block = '''	if !s.InBandFeeActive {
		if s.Config.EnableAsic && s.Protocol != "ETH_PROXY" {
			if !isExploit && (s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size || s.FeeExtranonce.En1 != s.MainExtranonce.En1) {
				extranonceToSend = s.MainExtranonce
				s.LastExtranonceCmdTime = time.Now()
			}
			
			s.mu.Lock()
			localDiff := s.LocalDiff
			s.mu.Unlock()
			if localDiff > 0 {
				difficultyToSend = localDiff
			}
		}'''

code = code.replace(old_stop_block, new_stop_block)


# 2. Fix readMainLoop chaotic forwarding
old_main_forward = '''			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if isExploit || inBandFeeActive {
					safeFprintf(minerConn, 5*time.Second, "%s\\n", line)
				} else {'''

new_main_forward = '''			if state == "FEE" || state == "SWITCHING_TO_FEE" {
				if inBandFeeActive {
					safeFprintf(minerConn, 5*time.Second, "%s\\n", line)
				} else {'''

code = code.replace(old_main_forward, new_main_forward)

# 3. Remove the buggy clean_jobs pendingDiff logic from readMainLoop and readFeeLoop
# It looks like:
#						// Flush pending difficulty if any
#						pendingDiff := s.PendingDiff
#						if pendingDiff > 0 && pendingDiff != s.LocalDiff && isCleanJobs {
#							s.LocalDiff = pendingDiff
#							s.PendingDiff = 0
#							if s.MinerConn != nil {
#								setDiffPkt := fmt.Sprintf({"id": null, "method": "mining.set_difficulty", "params": [%.0f]}+"\n", pendingDiff)
#								safeFprintf(s.MinerConn, 5*time.Second, "%s", setDiffPkt)
#							}
#						}

code = re.sub(r'(\t+// Flush pending difficulty if any\n\t+pendingDiff := s\.PendingDiff\n\t+if pendingDiff > 0 && pendingDiff != s\.LocalDiff && isCleanJobs \{\n\t+s\.LocalDiff = pendingDiff\n\t+s\.PendingDiff = 0\n\t+if s\.MinerConn != nil \{\n\t+setDiffPkt := fmt\.Sprintf\(\{"id": null, "method": "mining\.set_difficulty", "params": \[%\.0f\]\}\+"\\n", pendingDiff\)\n\t+safeFprintf\(s\.MinerConn, 5\*time\.Second, "%s", setDiffPkt\)\n\t+\}\n\t+\}\n)', '', code)


with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(code)

print("session.go updated!")
