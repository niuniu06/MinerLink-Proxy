import re

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'

with open(pcap_file, 'rb') as f:
    data = f.read()

strings = re.findall(b'[\x20-\x7E]{15,}', data)

print("Searching around id: 21")
for i, s in enumerate(strings):
    text = s.decode('ascii', errors='ignore')
    if '"id":21' in text or '"id": 21' in text:
        # print the few messages before and after
        start = max(0, i-5)
        end = min(len(strings), i+5)
        for j in range(start, end):
            print(strings[j].decode('ascii', errors='ignore')[:150])
        print("="*50)
