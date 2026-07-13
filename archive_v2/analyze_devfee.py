import json
from scapy.all import rdpcap, TCP, IP

print('Loading pcap...')
packets = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')

main_pool_ip = '18.167.64.28'
main_pool_port = 2288
devfee_pool_ip = '172.65.196.131'
devfee_pool_port = 1315

main_jobs = set()
devfee_jobs = set()

miner_states = {} # miner_ip_port -> {'status': 'connected', 'current_pool': None, 'disconnects': 0, 'events': []}

def get_miner_id(ip, port):
    return f"{ip}:{port}"

for pkt in packets:
    if not (pkt.haslayer(TCP) and pkt.haslayer(IP)):
        continue
        
    src = pkt[IP].src
    dst = pkt[IP].dst
    sport = pkt[TCP].sport
    dport = pkt[TCP].dport
    time = pkt.time
    
    # Check for FIN or RST (disconnects)
    flags = pkt[TCP].flags
    if 'F' in flags or 'R' in flags:
        # Is it a miner disconnecting?
        if dport in [10690, 10691, 3333, 443, 10010]: # Miner to Proxy
            miner_id = get_miner_id(src, sport)
            if miner_id in miner_states:
                miner_states[miner_id]['events'].append((time, f"DISCONNECT (FIN/RST from miner)"))
        elif sport in [10690, 10691, 3333, 443, 10010]: # Proxy to Miner
            miner_id = get_miner_id(dst, dport)
            if miner_id in miner_states:
                miner_states[miner_id]['events'].append((time, f"DISCONNECT (FIN/RST from proxy)"))
                
    if not pkt[TCP].payload:
        continue
        
    raw = bytes(pkt[TCP].payload)
    try:
        payload = raw.decode('utf-8')
    except:
        continue
        
    for line in payload.split('\n'):
        line = line.strip()
        if not line:
            continue
        try:
            data = json.loads(line)
            method = data.get('method')
            
            # Record pool jobs
            if src == main_pool_ip and sport == main_pool_port and method == 'mining.notify':
                job_id = data.get('params', [])[0]
                main_jobs.add(job_id)
            elif src == devfee_pool_ip and sport == devfee_pool_port and method == 'mining.notify':
                job_id = data.get('params', [])[0]
                devfee_jobs.add(job_id)
                
            # Track proxy -> miner messages
            if sport in [10690, 10691, 3333, 443, 10010]: 
                miner_id = get_miner_id(dst, dport)
                if miner_id not in miner_states:
                    miner_states[miner_id] = {'current_pool': 'Unknown', 'events': []}
                
                if method == 'mining.notify':
                    job_id = data.get('params', [])[0]
                    clean = data.get('params', [])[8] if len(data.get('params', [])) > 8 else None
                    if job_id in main_jobs:
                        pool = 'MAIN'
                    elif job_id in devfee_jobs:
                        pool = 'DEVFEE'
                    else:
                        pool = 'UNKNOWN'
                        
                    if miner_states[miner_id]['current_pool'] != pool:
                        miner_states[miner_id]['events'].append((time, f"SWITCHED TO {pool}"))
                        miner_states[miner_id]['current_pool'] = pool
                    
                    # Log if clean=False right after pool switch or generally
                    # miner_states[miner_id]['events'].append((time, f"notify {pool} clean={clean}"))
                elif method == 'mining.set_difficulty':
                    diff = data.get('params', [])[0]
                    miner_states[miner_id]['events'].append((time, f"set_diff {diff}"))
        except:
            pass

# Summarize
for miner_id, state in miner_states.items():
    events = state['events']
    disconnects = [e for e in events if 'DISCONNECT' in e[1]]
    if len(disconnects) > 0:
        print(f"\n--- Miner {miner_id} ---")
        for t, msg in events:
            print(f"[{float(t):.2f}] {msg}")

