import re

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'

with open(pcap_file, 'rb') as f:
    data = f.read()

strings = re.findall(b'[\x20-\x7E]{15,}', data)

print("Looking for job_id 745e820")
found = False
for s in strings:
    text = s.decode('ascii', errors='ignore')
    if '745e820' in text:
        print(text[:200])
        found = True

if not found:
    print("Not found")
