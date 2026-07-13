import sys

with open('internal/proxy/server.go', 'r', encoding='utf-8') as f:
    content = f.read()

target = '''			// Flush port history every 30 minutes
			if ticks%30 == 0 {
				if totalMultiplier == 0 {
					// Fallback if no sessions
					totalMultiplier = 1.0 
				}
				s.RingBuffer.FlushToDB(s.Config.ListenPort, s.Config.CoinName, 30, totalMultiplier)
			}'''
repl = '''			// Flush port history every 5 minutes
			if ticks%5 == 0 {
				if totalMultiplier == 0 {
					// Fallback if no sessions
					totalMultiplier = 1.0 
				}
				s.RingBuffer.FlushToDB(s.Config.ListenPort, s.Config.CoinName, 5, totalMultiplier)
			}'''
content = content.replace(target, repl)

with open('internal/proxy/server.go', 'w', encoding='utf-8') as f:
    f.write(content)
