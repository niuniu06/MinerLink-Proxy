import sys

with open('internal/proxy/manager.go', 'r', encoding='utf-8') as f:
    content = f.read()

target = 'if sess.MinerConn != nil && sess.MinerConn.RemoteAddr().String() == ip {'
repl = 'if sess.ID == ip {'
content = content.replace(target, repl)

with open('internal/proxy/manager.go', 'w', encoding='utf-8') as f:
    f.write(content)
