import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = re.finditer(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        if 'mining.set_extranonce' in s:
            print(s)
    except:
        pass
