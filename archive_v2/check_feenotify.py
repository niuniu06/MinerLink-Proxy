import re
with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

match = re.search(r'func \(s \*Session\) readFeeLoop[\s\S]*?if method == "mining.notify"[\s\S]*?\}', code)
if match:
    print(match.group(0)[:800])
else:
    print("Not found")
