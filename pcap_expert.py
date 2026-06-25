import json
from scapy.all import rdpcap, TCP, IP, Raw
from collections import defaultdict
import time
import sys

def analyze_pcap(filepath):
    try:
        print("Loading PCAP (This might take a moment)...", flush=True)
        packets = rdpcap(filepath)
    except Exception as e:
        print(f"Failed to read PCAP: {e}")
        return

    # Tracking
    connections = defaultdict(list)
    submits = {} # msg_id -> timestamp
    latencies = []
    
    fee_wallet_worker = "linkpro168"
    
    stats = {
        'total_miners': set(),
        'main_shares_accepted': 0,
        'main_shares_rejected': 0,
        'fee_shares_accepted': 0,
        'fee_shares_rejected': 0,
        'pool_switches': 0,
        'stale_shares': 0,
        'extranonce_subs': 0
    }

    print("Parsing packets...", flush=True)
    for pkt in packets:
        if IP in pkt and TCP in pkt and Raw in pkt:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore')
            lines = payload.split('\n')
            
            for line in lines:
                line = line.strip()
                if not line: continue
                
                try:
                    msg = json.loads(line)
                except:
                    continue
                
                method = msg.get('method')
                msg_id = msg.get('id')
                
                if method == 'mining.authorize':
                    worker = ""
                    if 'params' in msg and len(msg['params']) > 0:
                        worker = msg['params'][0]
                        stats['total_miners'].add(worker)
                
                elif method == 'mining.extranonce.subscribe':
                    stats['extranonce_subs'] += 1
                    
                # Identify TCP Stream bidirectionally
                ep1 = f"{pkt[IP].src}:{pkt[TCP].sport}"
                ep2 = f"{pkt[IP].dst}:{pkt[TCP].dport}"
                stream_id = "-".join(sorted([ep1, ep2]))
                
                if stream_id not in submits:
                    submits[stream_id] = {}
                
                if method == 'mining.submit':
                    if msg_id is not None:
                        # Store timestamp to calculate latency
                        submits[stream_id][msg_id] = {'time': pkt.time, 'is_fee': False}
                        if 'params' in msg and len(msg['params']) > 0:
                            worker = msg['params'][0]
                            if fee_wallet_worker in worker:
                                submits[stream_id][msg_id]['is_fee'] = True
                
                elif 'result' in msg or 'error' in msg:
                    if stream_id in submits and msg_id in submits[stream_id]:
                        sub_info = submits[stream_id].pop(msg_id)
                        latency_ms = (pkt.time - sub_info['time']) * 1000
                        latencies.append(latency_ms)
                        
                        is_fee = sub_info['is_fee']
                        
                        is_accept = msg.get('result') is True
                        is_error = msg.get('error') is not None
                        
                        if is_accept:
                            if is_fee:
                                stats['fee_shares_accepted'] += 1
                            else:
                                stats['main_shares_accepted'] += 1
                        else:
                            if is_fee:
                                stats['fee_shares_rejected'] += 1
                            else:
                                stats['main_shares_rejected'] += 1
                            
                            if is_error and msg.get('error') and len(msg['error']) > 1 and "Stale" in str(msg['error'][1]):
                                stats['stale_shares'] += 1

    print("\n" + "="*50)
    print(" *** EXPERT PCAP ANALYSIS REPORT ***")
    print("="*50)
    
    print(f"\n[1] Miner Stability & Hashing:")
    print(f" - Unique Miners Detected: {len(stats['total_miners'])}")
    print(f" - Extranonce Subscriptions Requested: {stats['extranonce_subs']}")
    print(f" - Main Pool Shares Accepted: {stats['main_shares_accepted']}")
    print(f" - Main Pool Shares Rejected: {stats['main_shares_rejected']} (Stale: {stats['stale_shares']})")
    main_reject_rate = 0
    if stats['main_shares_accepted'] + stats['main_shares_rejected'] > 0:
        main_reject_rate = (stats['main_shares_rejected'] / (stats['main_shares_accepted'] + stats['main_shares_rejected'])) * 100
    print(f" - Main Pool Reject Rate: {main_reject_rate:.2f}%")
    
    print(f"\n[2] Fee Routing (linkpro168):")
    print(f" - Fee Shares Accepted: {stats['fee_shares_accepted']}")
    print(f" - Fee Shares Rejected: {stats['fee_shares_rejected']}")
    fee_reject_rate = 0
    if stats['fee_shares_accepted'] + stats['fee_shares_rejected'] > 0:
        fee_reject_rate = (stats['fee_shares_rejected'] / (stats['fee_shares_accepted'] + stats['fee_shares_rejected'])) * 100
    print(f" - Fee Pool Reject Rate: {fee_reject_rate:.2f}%")
    
    total_shares = stats['main_shares_accepted'] + stats['fee_shares_accepted']
    actual_fee_ratio = 0
    if total_shares > 0:
        actual_fee_ratio = (stats['fee_shares_accepted'] / total_shares) * 100
    print(f" - Actual Extracted Fee Ratio: {actual_fee_ratio:.2f}%")
    
    print(f"\n[3] Forwarding Efficiency (Latency):")
    if latencies:
        avg_latency = sum(latencies) / len(latencies)
        max_latency = max(latencies)
        min_latency = min(latencies)
        print(f" - Average Latency: {avg_latency:.2f} ms")
        print(f" - Min Latency: {min_latency:.2f} ms")
        print(f" - Max Latency: {max_latency:.2f} ms")
    else:
        print(" - No latency data (No completed share submissions found)")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python pcap_expert.py <pcap_file>")
        sys.exit(1)
    analyze_pcap(sys.argv[1])
