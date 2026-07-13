import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

pool_ips = set()
for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    
    if pkt[TCP].dport == 2288:
        pool_ips.add(pkt[IP].dst)
    elif pkt[TCP].sport == 2288:
        pool_ips.add(pkt[IP].src)

print(f"Found pool IPs on port 2288: {pool_ips}")

for pool_ip in pool_ips:
    for pkt in packets:
        if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
        if pkt[IP].src == pool_ip:
            raw = bytes(pkt[TCP].payload)
            try:
                payload = raw.decode('utf-8')
                for line in payload.split('\n'):
                    if 'mining.notify' in line:
                        msg = json.loads(line)
                        clean = msg.get('params', [])[8] if len(msg.get('params', [])) > 8 else None
                        print(f"[{pkt.time:.2f}] {pool_ip} Notify clean_jobs={clean}")
            except: pass

