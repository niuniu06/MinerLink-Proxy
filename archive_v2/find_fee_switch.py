import re
import json

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
for i, m in enumerate(matches):
    try:
        s = m.group(0).decode('utf-8')
        if "linkpro168" in s:
            print(f"--- MATCH AT {i} ---")
            for j in range(max(0, i-5), min(len(matches), i+5)):
                try:
                    print(f"{j}: {matches[j].group(0).decode('utf-8')[:150]}")
                except:
                    pass
    except:
        pass
