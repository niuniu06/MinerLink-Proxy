with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()

import re
matches = re.finditer(b'f2pool|linkpro168|btc-asia', data, re.IGNORECASE)
found = False
for m in matches:
    print(f"Found F2Pool string: {m.group(0)}")
    found = True
    break

if not found:
    print("No F2Pool traffic found in the PCAP.")
