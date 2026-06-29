import json
from scapy.all import rdpcap, TCP, IP

print('Loading pcap...')
packets = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')
print(f'Loaded {len(packets)} packets.')

for pkt in packets:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].payload:
        raw_payload = bytes(pkt[TCP].payload)
        try:
            payload_str = raw_payload.decode('utf-8')
        except UnicodeDecodeError:
            continue
            
        src = f'{pkt[IP].src}:{pkt[TCP].sport}'
        dst = f'{pkt[IP].dst}:{pkt[TCP].dport}'
        
        for line in payload_str.split('\n'):
            line = line.strip()
            if not line:
                continue
            try:
                data = json.loads(line)
                method = data.get('method')
                if method == 'mining.set_difficulty':
                    diff = data.get('params', [])
                    print(f'[{src} -> {dst}] set_difficulty: {diff}')
                elif method == 'mining.notify':
                    params = data.get('params', [])
                    clean_jobs = params[8] if len(params) > 8 else None
                    print(f'[{src} -> {dst}] notify, clean_jobs: {clean_jobs}')
            except json.JSONDecodeError:
                pass
