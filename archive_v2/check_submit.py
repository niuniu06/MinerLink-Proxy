import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

count = 0
matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        obj = json.loads(s)
        if obj.get('method') == 'mining.submit':
            print("SUBMIT:", obj.get('params')[0], obj.get('params')[1])
            count += 1
            if count > 20:
                break
    except:
        pass
