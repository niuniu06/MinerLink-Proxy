import re

with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

# 1. Fix mining.set_difficulty forwarding logic
old_set_diff = '''								s.mu.Lock()
								isFirstDiff := (s.LocalDiff == 0)
								s.CurrentDiff = diffFloat
								s.MainDifficulty = diffFloat
								s.RemoteDiff = diffFloat
								if !enableVardiff || s.LocalDiff == 0 {
									s.PendingDiff = diffFloat
								}
								s.mu.Unlock()
								GlobalDispatcher.UpdateDiff(s.Config.PoolAddress, diffFloat)

								if isFirstDiff {
									// For the very first difficulty of the session, we MUST forward it immediately
									// so the miner doesn't start hashing at diff 1 on the first notify.
									s.mu.Lock()
									s.LocalDiff = diffFloat
									s.PendingDiff = 0
									s.mu.Unlock()
									// Do NOT continue, let it fall through to shouldForward so it goes to the miner!
								} else {
									// Otherwise INTERCEPT!
									continue
								}'''

new_set_diff = '''								s.mu.Lock()
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

code = code.replace(old_set_diff, new_set_diff)

# 2. Delete probabilistic fake accept in isMainRoute block
old_fake_accept = '''					// Vardiff Fake Accept logic check
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


					if shouldFakeAccept {
						if id, ok := msg["id"]; ok {
							s.pendingShares.Delete(id)
							fakeReply := fmt.Sprintf({"id": %v, "result": true, "error": null}+"\\n", id)
							safeWrite(s.MinerConn, []byte(fakeReply), 5*time.Second)
						}
					} else {
						if id, ok := msg["id"]; ok {
							s.pendingShares.Store(id, PendingShare{Req: strings.TrimSpace(line), IsFee: false, FeeMode: FeeModeNone})
						}
						safeFprintf(mainConn, 5*time.Second, "%s\\n", line)
					}'''

new_fake_accept = '''					if id, ok := msg["id"]; ok {
						s.pendingShares.Store(id, PendingShare{Req: strings.TrimSpace(line), IsFee: false, FeeMode: FeeModeNone})
					}
					safeFprintf(mainConn, 5*time.Second, "%s\\n", line)'''

code = code.replace(old_fake_accept, new_fake_accept)

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(code)

print("session.go updated!")
