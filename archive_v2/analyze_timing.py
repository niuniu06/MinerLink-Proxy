import json
from scapy.all import rdpcap, TCP, IP

packets = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')

diffs = []

for pkt in packets:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload:
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
        except:
            continue
            
        src_ip = pkt[IP].src
        sport = pkt[TCP].sport
        time = pkt.time
        
        for line in payload.split('\n'):
            line = line.strip()
            if not line:
                continue
            try:
                data = json.loads(line)
                method = data.get('method')
                if method == 'mining.set_difficulty':
                    diff = data.get('params', [])[0]
                    direction = 'Proxy->Miner' if sport in [10010, 3333, 443, 10690, 10691] else 'Pool->Proxy'
                    diffs.append((time, direction, diff))
            except:
                pass

# Sort by time just in case
diffs.sort(key=lambda x: x[0])

for i, (t, d, v) in enumerate(diffs[:40]):
    print(f'[{float(t):.4f}] {d}: {v}')
