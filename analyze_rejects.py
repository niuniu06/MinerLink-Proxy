import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

miners_found = set()
for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    src = pkt[IP].src
    dst = pkt[IP].dst
    sport = pkt[TCP].sport
    dport = pkt[TCP].dport
    
    if dport == 10690:
        miners_found.add(f"{src}:{sport}")
        
print(list(miners_found)[:5])

target_miner = list(miners_found)[0] if miners_found else None

if target_miner:
    target_ip, target_port = target_miner.split(':')
    target_port = int(target_port)
    print(f"\nAnalyzing {target_miner}...")
    for pkt in packets:
        if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
        src = pkt[IP].src
        dst = pkt[IP].dst
        sport = pkt[TCP].sport
        dport = pkt[TCP].dport
        
        if (src == target_ip and sport == target_port) or (dst == target_ip and dport == target_port):
            raw = bytes(pkt[TCP].payload)
            try:
                payload = raw.decode('utf-8')
                for line in payload.split('\n'):
                    if line.strip() and 'mining.submit' not in line:
                        print(f"[{pkt.time:.2f}] {src}:{sport} -> {dst}:{dport}: {line.strip()[:150]}")
            except: pass

