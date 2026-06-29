import json
from scapy.all import rdpcap, TCP, IP

packets = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')

# Identify roles based on ports
# Proxy listens on 10010 usually, or 3333.
# Let's just look at the json method and source/dest port.

pool_to_proxy_diffs = 0
proxy_to_miner_diffs = 0

for pkt in packets:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload:
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
        except:
            continue
            
        src_ip = pkt[IP].src
        dst_ip = pkt[IP].dst
        sport = pkt[TCP].sport
        dport = pkt[TCP].dport
        
        for line in payload.split('\n'):
            line = line.strip()
            if not line:
                continue
            try:
                data = json.loads(line)
                method = data.get('method')
                if method == 'mining.set_difficulty':
                    diff = data.get('params', [])[0]
                    # If destination port is a known proxy listen port, it's miner->proxy (impossible for set_difficulty)
                    # If source port is a proxy listen port, it's proxy->miner
                    if sport in [10010, 3333, 443, 10690, 10691]:
                        print(f'[Proxy {src_ip}:{sport} -> Miner {dst_ip}:{dport}] set_difficulty: {diff}')
                        proxy_to_miner_diffs += 1
                    else:
                        print(f'[Pool {src_ip}:{sport} -> Proxy {dst_ip}:{dport}] set_difficulty: {diff}')
                        pool_to_proxy_diffs += 1
            except:
                pass

print(f'\nTotal Pool -> Proxy set_difficulty: {pool_to_proxy_diffs}')
print(f'Total Proxy -> Miner set_difficulty: {proxy_to_miner_diffs}')
