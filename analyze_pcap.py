import re

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'

with open(pcap_file, 'rb') as f:
    data = f.read()

# Extract printable ASCII sequences (length > 10)
strings = re.findall(b'[\x20-\x7E]{10,}', data)

for s in strings:
    text = s.decode('ascii', errors='ignore')
    if 'result":false' in text.replace(' ', '') or 'result": false' in text or 'error":[' in text.replace(' ', ''):
        print("Found error/reject:")
        print(text)
        print("-" * 40)
