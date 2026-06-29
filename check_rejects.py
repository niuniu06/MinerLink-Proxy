import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    src = pkt[IP].src
    dst = pkt[IP].dst
    sport = pkt[TCP].sport
    dport = pkt[TCP].dport
    
    if dst == '36.45.254.81' and sport == 10690:
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
            for line in payload.split('\n'):
                if 'result":false' in line.replace(' ', '') or 'error":[' in line.replace(' ', ''):
                    print(f"[{pkt.time:.2f}] Proxy -> Miner {dport}: {line.strip()[:150]}")
        except: pass

