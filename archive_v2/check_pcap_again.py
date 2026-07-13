import json
import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = re.finditer(b'\{.*?\}', data)
error_count = 0
for m in matches:
    try:
        s = m.group(0).decode('utf-8')
        if '"error":[' in s or '"error": [' in s:
            error_count += 1
    except:
        pass
print("Total errors found:", error_count)
