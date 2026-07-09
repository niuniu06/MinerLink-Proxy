$content = Get-Content -Path "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go" -Raw

# 1. Add RingBuffer to Session struct
$content = $content -replace 'Stats          SessionStats\s*\r?\n\s*quit', "Stats          SessionStats`n`tRingBuffer     *HashrateRingBuffer`n`tquit"

# 2. Init RingBuffer in NewSession
$content = $content -replace 'Stats:          SessionStats\{ConnectedAt: time\.Now\(\)\},\s*\r?\n\s*quit:', "Stats:          SessionStats{ConnectedAt: time.Now()},`n`t`tRingBuffer:     &HashrateRingBuffer{},`n`t`tquit:"

# 3. Add AddShare hook in 3 places
$content = $content -replace 's\.ShareHistory = append\(s\.ShareHistory, ShareEvent\{\s*\r?\n\s*Timestamp: time\.Now\(\),\s*\r?\n\s*Diff:\s*s\.CurrentDiff,\s*\r?\n\s*\}\)', "s.ShareHistory = append(s.ShareHistory, ShareEvent{`n`t`t`t`t`t`t`t`tTimestamp: time.Now(),`n`t`t`t`t`t`t`t`tDiff:      s.CurrentDiff,`n`t`t`t`t`t`t`t})`n`t`t`t`t`t`t`t`n`t`t`t`t`t`t`tif isFee {`n`t`t`t`t`t`t`t`ts.RingBuffer.AddShare(s.CurrentDiff, true)`n`t`t`t`t`t`t`t`tif s.Server != nil { s.Server.RingBuffer.AddShare(s.CurrentDiff, true) }`n`t`t`t`t`t`t`t} else {`n`t`t`t`t`t`t`t`ts.RingBuffer.AddShare(s.CurrentDiff, false)`n`t`t`t`t`t`t`t`tif s.Server != nil { s.Server.RingBuffer.AddShare(s.CurrentDiff, false) }`n`t`t`t`t`t`t`t}"

# 4. Add ONLINE hook when restored
$content = $content -replace 's\.LogGeneral\("Miner session restored from offline state, inherited %d valid shares", oldStats\.ValidShares\)', "s.LogGeneral(`"Miner session restored from offline state, inherited %d valid shares`", oldStats.ValidShares)`n`t`t`t`t`t`t`tdb.DB.Create(&models.EventLog{`n`t`t`t`t`t`t`t`tTimestamp:   time.Now(),`n`t`t`t`t`t`t`t`tMinerIP:     s.MinerConn.RemoteAddr().String(),`n`t`t`t`t`t`t`t`tMinerWorker: s.MinerWorker,`n`t`t`t`t`t`t`t`tEventType:   `"ONLINE`",`n`t`t`t`t`t`t`t`tMessage:     fmt.Sprintf(`"Miner restored: %s`", s.MinerWorker),`n`t`t`t`t`t`t`t})"

# 5. Add ONLINE hook when authorized
$content = $content -replace 's\.LogGeneral\("Miner authorized: %s", s\.MinerWorker\)', "s.LogGeneral(`"Miner authorized: %s`", s.MinerWorker)`n`t`t`t`t`t`t`tdb.DB.Create(&models.EventLog{`n`t`t`t`t`t`t`t`tTimestamp:   time.Now(),`n`t`t`t`t`t`t`t`tMinerIP:     s.MinerConn.RemoteAddr().String(),`n`t`t`t`t`t`t`t`tMinerWorker: s.MinerWorker,`n`t`t`t`t`t`t`t`tEventType:   `"ONLINE`",`n`t`t`t`t`t`t`t`tMessage:     fmt.Sprintf(`"Miner connected: %s`", s.MinerWorker),`n`t`t`t`t`t`t`t})"

# 6. Add OFFLINE hook in Close
$content = $content -replace 'func \(s \*Session\) Close\(\) \{\s*\r?\n\s*s\.LogGeneral\("Session Close called"\)\s*\r?\n\s*s\.mu\.Lock\(\)\s*\r?\n\s*defer s\.mu\.Unlock\(\)\s*\r?\n\s*\r?\n\s*select \{\s*\r?\n\s*case <-s\.quit:\s*\r?\n\s*return\s*\r?\n\s*default:\s*\r?\n\s*close\(s\.quit\)\s*\r?\n\s*\}\s*\r?\n\s*\r?\n\s*if s\.MinerConn != nil \{', "func (s *Session) Close() {`n`ts.LogGeneral(`"Session Close called`")`n`ts.mu.Lock()`n`tdefer s.mu.Unlock()`n`n`tselect {`n`tcase <-s.quit:`n`t`treturn`n`tdefault:`n`t`tclose(s.quit)`n`t}`n`n`tif s.MinerWorker != `"`" {`n`t`tdb.DB.Create(&models.EventLog{`n`t`t`tTimestamp:   time.Now(),`n`t`t`tMinerIP:     s.MinerConn.RemoteAddr().String(),`n`t`t`tMinerWorker: s.MinerWorker,`n`t`t`tEventType:   `"OFFLINE`",`n`t`t`tMessage:     fmt.Sprintf(`"Miner disconnected: %s`", s.MinerWorker),`n`t`t})`n`t}`n`n`tif s.MinerConn != nil {"

$content | Set-Content -Path "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go" -Encoding UTF8
