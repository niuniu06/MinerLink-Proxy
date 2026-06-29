import re

with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

matches = list(re.finditer(b'\{.*?\}', data))
print(f"Total JSON packets: {len(matches)}")
for m in matches[-50:]:
    try:
        s = m.group(0).decode('utf-8')
        print(s[:100])
    except:
        pass
