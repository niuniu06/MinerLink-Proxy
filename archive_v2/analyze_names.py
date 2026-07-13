import json
from scapy.all import rdpcap, TCP, IP

packets = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')

miner_info = {} # port -> {'worker': None, 'agent': None}

for pkt in packets:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload:
        raw = bytes(pkt[TCP].payload)
        try:
            payload = raw.decode('utf-8')
        except:
            continue
            
        src_ip = pkt[IP].src
        sport = pkt[TCP].sport
        
        if src_ip == '36.45.254.81':
            if sport not in miner_info:
                miner_info[sport] = {'worker': 'Unknown', 'agent': 'Unknown'}
                
            for line in payload.split('\n'):
                line = line.strip()
                if not line:
                    continue
                try:
                    data = json.loads(line)
                    method = data.get('method')
                    if method == 'mining.authorize':
                        worker = data.get('params', [])[0]
                        miner_info[sport]['worker'] = worker
                    elif method == 'mining.subscribe':
                        agent = data.get('params', [])[0]
                        miner_info[sport]['agent'] = agent
                except:
                    pass

for port, info in miner_info.items():
    print(f"Port {port}: Worker='{info['worker']}', Agent='{info['agent']}'")
