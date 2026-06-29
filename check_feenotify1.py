import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        if 'mining.notify' in s and 'BBvRjEdFO' in s:
            print("NOTIFY:", s)
            break
    except:
        pass
