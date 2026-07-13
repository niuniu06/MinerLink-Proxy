with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

import re
old_set_diff_regex = r's\.mu\.Lock\(\)\n\s+isFirstDiff := \(s\.LocalDiff == 0\)\n\s+s\.CurrentDiff = diffFloat\n\s+s\.MainDifficulty = diffFloat\n\s+s\.RemoteDiff = diffFloat\n\s+if !enableVardiff \|\| s\.LocalDiff == 0 \{\n\s+s\.PendingDiff = diffFloat\n\s+\}\n\s+s\.mu\.Unlock\(\)\n\s+GlobalDispatcher\.UpdateDiff\(s\.Config\.PoolAddress, diffFloat\)\n\n\s+if isFirstDiff \{\n\s+// For the very first difficulty of the session, we MUST forward it immediately\n\s+// so the miner doesn\'t start hashing at diff 1 on the first notify\.\n\s+s\.mu\.Lock\(\)\n\s+s\.LocalDiff = diffFloat\n\s+s\.PendingDiff = 0\n\s+s\.mu\.Unlock\(\)\n\s+// Do NOT continue, let it fall through to shouldForward so it goes to the miner!\n\s+\} else \{\n\s+// Otherwise INTERCEPT!\n\s+continue\n\s+\}'

new_set_diff = '''s.mu.Lock()
								isFirstDiff := (s.LocalDiff == 0)
								s.CurrentDiff = diffFloat
								s.MainDifficulty = diffFloat
								s.RemoteDiff = diffFloat
								
								// [VarDiff Fix] We MUST enforce LocalDiff >= RemoteDiff. 
								// If the pool asks for a higher difficulty than we are currently mining at,
								// we MUST immediately adopt it to prevent the pool from rejecting our shares!
								forceUpdateLocal := false
								if !enableVardiff || s.LocalDiff == 0 || diffFloat > s.LocalDiff {
									s.PendingDiff = diffFloat
									forceUpdateLocal = true
								}
								s.mu.Unlock()
								GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

								if forceUpdateLocal {
									s.mu.Lock()
									s.LocalDiff = diffFloat
									s.PendingDiff = 0
									s.mu.Unlock()
									safeFprintf(s.MinerConn, 5*time.Second, "%s\\n", line)
									continue
								} else {
									// If enableVardiff is true and diffFloat <= LocalDiff,
									// we intercept it. VarDiff is maintaining a higher or equal difficulty.
									continue
								}'''

code = re.sub(old_set_diff_regex, new_set_diff, code)

old_fake_accept_regex = r'// Vardiff Fake Accept logic check[\s\S]*?if shouldFakeAccept \{\n\s+if id, ok := msg\["id"\]; ok \{\n\s+s\.pendingShares\.Delete\(id\)\n\s+fakeReply := fmt\.Sprintf\(\{"id": %v, "result": true, "error": null\}\+"\\n", id\)\n\s+safeWrite\(s\.MinerConn, \[\]byte\(fakeReply\), 5\*time\.Second\)\n\s+\}\n\s+\} else \{\n\s+if id, ok := msg\["id"\]; ok \{\n\s+s\.pendingShares\.Store\(id, PendingShare\{Req: strings\.TrimSpace\(line\), IsFee: false, FeeMode: FeeModeNone\}\)\n\s+\}\n\s+safeFprintf\(mainConn, 5\*time\.Second, "%s\\n", line\)\n\s+\}'

new_fake_accept = '''if id, ok := msg["id"]; ok {
						s.pendingShares.Store(id, PendingShare{Req: strings.TrimSpace(line), IsFee: false, FeeMode: FeeModeNone})
					}
					safeFprintf(mainConn, 5*time.Second, "%s\\n", line)'''

code = re.sub(old_fake_accept_regex, new_fake_accept, code)

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(code)

print("session.go updated!")
