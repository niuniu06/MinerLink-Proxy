# -*- coding: utf-8 -*-
with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Fix 1: readMinerLoop AddShare
target1 = '''							s.RingBuffer.AddShare(s.CurrentDiff, isFee)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, isFee)
							}'''
repl1 = '''							isDevFee := pending.FeeMode == FeeModeDev
							s.RingBuffer.AddShare(s.CurrentDiff, isFee, isDevFee)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, isFee, isDevFee)
							}'''
content = content.replace(target1, repl1)

# Fix 2: readFeeLoop AddShare (Submit success)
target2 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, true)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true)
							}
							s.mu.Unlock()
						}

						if isReject {'''
repl2 = '''							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							}
							s.mu.Unlock()
						}

						if isReject {'''
content = content.replace(target2, repl2)

# Fix 3: readFeeLoop AddShare (Submit rejected but masked)
target3 = '''							// Always append to history to keep UI Hashrate stable
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, true)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true)
							}

							if isDevMode {'''
repl3 = '''							// Always append to history to keep UI Hashrate stable
							s.ShareHistory = append(s.ShareHistory, ShareEvent{
								Timestamp: time.Now(),
								Diff:      s.CurrentDiff,
							})
							s.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							if s.Server != nil {
								s.Server.RingBuffer.AddShare(s.CurrentDiff, true, isDevMode)
							}

							if isDevMode {'''
content = content.replace(target3, repl3)


with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(content)
