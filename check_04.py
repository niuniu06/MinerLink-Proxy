import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

count = 0
matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        if 'S21-04' in s:
            count += 1
    except:
        pass
print("S21-04 packets:", count)
