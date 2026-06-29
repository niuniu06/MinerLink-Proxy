import json
from scapy.all import rdpcap, TCP, IP

print('Loading new pcap...')
try:
    packets = rdpcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap')
except Exception as e:
    print(f"Failed to load pcap: {e}")
    exit(1)
print(f'Loaded {len(packets)} packets.')

main_pool_ip = '18.167.64.28'
main_pool_port = 2288
devfee_pool_ip = '172.65.196.131'
devfee_pool_port = 1315

miner_info = {} # ip:port -> {worker, agent, disconnects, set_diffs, switches, connect_time}
pool_diffs = [] # (time, pool, diff)
proxy_diffs = [] # (time, miner, diff)

for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP)): continue
    src = pkt[IP].src
    dst = pkt[IP].dst
    sport = pkt[TCP].sport
    dport = pkt[TCP].dport
    time = pkt.time
    
    # Miner identifier
    miner_id = None
    if dport in [10690, 10691, 3333, 443, 10010, 10000, 1111]: # Miner -> Proxy
        miner_id = f"{src}:{sport}"
    elif sport in [10690, 10691, 3333, 443, 10010, 10000, 1111]: # Proxy -> Miner
        miner_id = f"{dst}:{dport}"
        
    if miner_id and miner_id not in miner_info:
        miner_info[miner_id] = {'worker': 'Unknown', 'agent': 'Unknown', 'disconnects': 0, 'set_diffs': 0, 'switches': 0, 'connect_time': time, 'last_seen': time}
        
    if miner_id:
        miner_info[miner_id]['last_seen'] = time
        
    flags = pkt[TCP].flags
    if ('F' in flags or 'R' in flags) and miner_id:
        miner_info[miner_id]['disconnects'] += 1
        
    if not pkt[TCP].payload: continue
    raw = bytes(pkt[TCP].payload)
    try:
        payload = raw.decode('utf-8')
    except:
        continue
        
    for line in payload.split('\n'):
        line = line.strip()
        if not line: continue
        try:
            data = json.loads(line)
            method = data.get('method')
            
            if method == 'mining.subscribe' and miner_id:
                if 'params' in data and len(data['params']) > 0:
                    miner_info[miner_id]['agent'] = data['params'][0]
            elif method == 'mining.authorize' and miner_id:
                if 'params' in data and len(data['params']) > 0:
                    miner_info[miner_id]['worker'] = data['params'][0]
            elif method == 'mining.set_difficulty':
                diff = data.get('params', [])[0]
                if src in [main_pool_ip, devfee_pool_ip]:
                    pool = 'MAIN' if src == main_pool_ip else 'DEVFEE'
                    pool_diffs.append((time, pool, diff))
                elif miner_id and sport in [10690, 10691, 3333, 443, 10010, 10000, 1111]:
                    proxy_diffs.append((time, miner_id, diff))
                    miner_info[miner_id]['set_diffs'] += 1
        except:
            pass

print(f"\n--- Miner Summary ---")
for mid, info in miner_info.items():
    duration = info['last_seen'] - info['connect_time']
    print(f"{mid} [{info['agent'][:20]}] {info['worker']} - duration: {float(duration):.1f}s, diff_changes: {info['set_diffs']}, disconnects: {info['disconnects']}")

print(f"\n--- Pool->Proxy Diffs ---")
for i, (t, p, d) in enumerate(pool_diffs[:15]):
    print(f"[{float(t):.2f}] {p} -> Proxy: {d}")
    
print(f"\n--- Proxy->Miner Diffs ---")
for i, (t, m, d) in enumerate(proxy_diffs[:15]):
    print(f"[{float(t):.2f}] Proxy -> Miner {m}: {d}")

