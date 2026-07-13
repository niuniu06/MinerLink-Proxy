import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    src = pkt[IP].src
    
    if not src.startswith('172.31') and not src.startswith('36.45') and not src.startswith('192.168'):
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
            for line in payload.split('\n'):
                if 'mining.notify' in line:
                    print(f"[{pkt.time:.2f}] {src} -> Proxy: {line.strip()[:150]}")
        except: pass

