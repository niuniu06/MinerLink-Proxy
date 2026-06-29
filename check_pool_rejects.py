import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    src = pkt[IP].src
    
    if src == '18.167.64.28':
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
            for line in payload.split('\n'):
                if '"result":' in line and ('false' in line.lower() or 'null' in line.lower() or 'error' in line.lower()):
                    print(f"[{pkt.time:.2f}] Pool -> Proxy: {line.strip()[:150]}")
        except: pass

