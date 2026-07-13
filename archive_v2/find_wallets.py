import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

wallets = set()
matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        obj = json.loads(s)
        if obj.get('method') == 'mining.submit':
            params = obj.get('params', [])
            if len(params) > 0 and isinstance(params[0], str):
                wallets.add(params[0])
    except:
        pass

for w in wallets:
    print(w)
