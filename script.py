import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
for m in matches: 
    try:
        j = json.loads(m.group(0).decode('utf-8', 'ignore'))
        if 'method' in j and j['method'] == 'mining.notify' and j['params'][0] == '2116636':
            print(f'Found notify: {j}')
    except:
        pass
