import sys

with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    content = f.read()

target1 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})'''
repl1 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, isFee)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, isFee)
							}'''

target2 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.mu.Unlock()
						}
					}
				}
				
				// Check for Fee Pool intercepts'''
repl2 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, true)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true)
							}
							s.mu.Unlock()
						}
					}
				}
				
				// Check for Fee Pool intercepts'''

content = content.replace(target1, repl1)
content = content.replace(target2, repl2)

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(content)
