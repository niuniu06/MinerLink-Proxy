from scapy.all import rdpcap, TCP
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
fee_streams = set()

for p in packets:
    if TCP in p:
        payload = bytes(p[TCP].payload)
        if b'linkpro168' in payload:
            fee_streams.add((p[TCP].sport, p[TCP].dport))
            fee_streams.add((p[TCP].dport, p[TCP].sport))

all_submits = 0
fee_submits = 0

for p in packets:
    if TCP in p:
        payload = bytes(p[TCP].payload)
        if b'mining.submit' in payload:
            all_submits += 1
            if (p[TCP].sport, p[TCP].dport) in fee_streams:
                fee_submits += 1

print(f"Total mining.submit seen in pcap: {all_submits}")
print(f"Fee mining.submit seen in pcap: {fee_submits}")
