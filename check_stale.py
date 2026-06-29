import json

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

import re
matches = re.finditer(b'\{.*?\}', data)
submit_map = {}
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        obj = json.loads(s)
        if obj.get('method') == 'mining.submit':
            submit_map[obj.get('id')] = obj.get('params')
        elif 'error' in obj and obj.get('error') is not None:
            err = obj.get('error')
            if err[0] == 23:
                print(f"Rejected ID {obj.get('id')} with job {submit_map.get(obj.get('id'))}")
    except:
        pass
