import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
for m in matches[:20]:
    try:
        s = m.group(0).decode('utf-8')
        print(s)
    except:
        pass
