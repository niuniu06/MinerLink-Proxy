import re
with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

match = re.search(r'func \(s \*Session\) readFeeLoop[\s\S]*?\}', code)
if match:
    print(match.group(0)[:2000])
else:
    print("Not found")
