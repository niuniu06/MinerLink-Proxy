import re

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'

with open(pcap_file, 'rb') as f:
    data = f.read()

strings = re.findall(b'[\x20-\x7E]{15,}', data)

print("Analyzing JSON responses in PCAP...")
count = 0
for s in strings:
    text = s.decode('ascii', errors='ignore')
    if 'result' in text and 'id' in text:
        # just print first 50 chars to keep it brief
        print(text[:100])
        count += 1
        if count > 30:
            break
