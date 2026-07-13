import json
with open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'r', encoding='utf-8') as f:
    content = f.read()

print("Original code snippet for mining.notify in readMainLoop:")
start = content.find('} else if method == "mining.notify" {')
print(content[start:start+1000])
