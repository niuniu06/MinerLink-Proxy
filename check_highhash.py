import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
for i, m in enumerate(matches):
    try:
        s = m.group(0).decode('utf-8')
        if 'high-hash' in s:
            print("--- REJECT ---")
            for j in range(max(0, i-5), i+1):
                try:
                    print(matches[j].group(0).decode('utf-8'))
                except:
                    pass
            break
    except:
        pass
