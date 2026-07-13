import json

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

import re
matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        obj = json.loads(s)
        if obj.get('method') == 'mining.notify':
            print("NOTIFY:", obj.get('params')[0], obj.get('params')[8])
    except:
        pass
