with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

code = code.replace('+isExploit := s.IsF2PoolExploit', '\tisExploit := s.IsF2PoolExploit')
code = code.replace('+s.mu.Unlock()', '\ts.mu.Unlock()')

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(code)

print("session.go fixed!")
