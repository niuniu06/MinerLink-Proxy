import os
import re

# Fix server.go
path = r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\server.go'
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# Add back close(s.Quit) if missing
if 'close(s.Quit)' not in content:
    content = content.replace('if s.FeeScheduler != nil', 'close(s.Quit)\n\tif s.FeeScheduler != nil')

# Suppress connection and accept errors from main log
content = re.sub(r'log\.Printf\(\"Accept error on port.*?\)','// removed accept error log', content)
content = re.sub(r'log\.Printf\(\"Incoming ZSDT Tunnel.*?\)','// removed incoming tunnel log', content)
content = re.sub(r'log\.Printf\(\"Tunnel TLS handshake failed.*?\)','// removed TLS error log', content)
content = re.sub(r'log\.Printf\(\"Tunnel Yamux server failed.*?\)','// removed Yamux error log', content)
content = re.sub(r'log\.Printf\(\"Tunnel session closed.*?\)','// removed session closed log', content)

with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

# Fix scheduler.go
path = r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\scheduler.go'
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()
content = re.sub(r'log\.Printf\(\"Starting Stateless Distributed.*?\)','// removed scheduler startup log', content)
content = re.sub(r'log\.Printf\(\"\[Scheduler\] Miner.*?\)','// removed scheduler log', content)
with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

# Fix vardiff.go
path = r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\vardiff.go'
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()
content = re.sub(r'log\.Printf\(\"\[Vardiff\] Miner.*?\)','// removed vardiff log', content)
with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

# Fix session.go LogGeneral and LogBackend to avoid any direct log.Printf output except via API
path = r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go'
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()
# Find LogGeneral and modify it so it doesn't write to global log, but only to MinerLogger
# Wait, LogGeneral ALREADY only writes to MinerLogger!
# But there might be other log.Printf calls in session.go.
content = re.sub(r'log\.Printf\(\"\[Watchdog\] Miner.*?\)','// removed watchdog log', content)
with open(path, 'w', encoding='utf-8') as f:
    f.write(content)

print("Logs cleaned successfully.")
