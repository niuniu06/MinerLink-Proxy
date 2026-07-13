import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

print(f"Pcap duration: {packets[-1].time - packets[0].time} seconds")
print(f"Total packets: {len(packets)}")
