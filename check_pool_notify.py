import json
from scapy.all import rdpcap, TCP, IP

try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    exit(1)

main_pool_ip = '18.167.64.28'
for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload): continue
    src = pkt[IP].src
    
    if src == main_pool_ip:
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
            for line in payload.split('\n'):
                if 'mining.notify' in line:
                    try:
                        msg = json.loads(line)
                        clean_jobs = msg['params'][8]
                        print(f"[{pkt.time:.2f}] Pool Notify clean_jobs={clean_jobs}: {line.strip()[:100]}")
                    except: pass
        except: pass

