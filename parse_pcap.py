import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
start_idx = 3600
end_idx = min(len(matches), start_idx + 25)

for i in range(start_idx, end_idx):
    try:
        print(f"{i}: {matches[i].group(0).decode('utf-8')[:200]}")
    except:
        pass
