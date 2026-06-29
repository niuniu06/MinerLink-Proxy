import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        obj = json.loads(s)
        if obj.get('method') == 'mining.submit':
            params = obj.get('params', [])
            if len(params) > 0 and isinstance(params[0], str) and 'linkpro168' in params[0]:
                print(f"Fee Submit found: {s}")
    except:
        pass
