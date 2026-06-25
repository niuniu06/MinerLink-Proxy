import json
from scapy.all import rdpcap, TCP, IP, Raw
from collections import defaultdict
import time
import sys

def analyze_pcap(filepath):
    try:
        print("Loading PCAP...", flush=True)
        packets = rdpcap(filepath)
    except Exception as e:
        print(f"Error loading pcap: {e}")
        return

    print(f"Total packets loaded: {len(packets)}")
    
    connections = defaultdict(lambda: {"syn": 0, "fin": 0, "rst": 0, "payloads": []})
    
    submits = 0
    accepts = 0
    rejects = 0
    
    fee_submits = 0
    
    latencies = []
    
    # Track submit messages by message ID to calculate latency
    pending_submits = {}
    
    for pkt in packets:
        if IP in pkt and TCP in pkt:
            src = pkt[IP].src
            dst = pkt[IP].dst
            sport = pkt[TCP].sport
            dport = pkt[TCP].dport
            flags = pkt[TCP].flags
            
            flow_key = f"{src}:{sport}-{dst}:{dport}"
            rev_key = f"{dst}:{dport}-{src}:{sport}"
            
            # Simple connection tracking
            if "S" in flags:
                connections[src]["syn"] += 1
            if "F" in flags:
                connections[src]["fin"] += 1
            if "R" in flags:
                connections[src]["rst"] += 1
                
            if Raw in pkt:
                try:
                    payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
                    for line in payload.split('\n'):
                        line = line.strip()
                        if not line:
                            continue
                        try:
                            msg = json.loads(line)
                            
                            # Stratum Submit
                            if msg.get("method") == "mining.submit":
                                submits += 1
                                msg_id = msg.get("id")
                                worker = msg.get("params", [""])[0] if msg.get("params") else ""
                                
                                # Check if it's a devfee submit (usually high ID or specific wallet name)
                                # Assuming devfee worker might have "duanjunli" or "1x22" or high ID
                                if "duanjunli" in worker or int(msg_id) > 90000:
                                    fee_submits += 1
                                    
                                pending_submits[msg_id] = pkt.time
                                
                            # Stratum Response (Accept/Reject)
                            elif "result" in msg and "error" in msg:
                                msg_id = msg.get("id")
                                if msg_id in pending_submits:
                                    latency = float(pkt.time - pending_submits[msg_id])
                                    latencies.append(latency)
                                    del pending_submits[msg_id]
                                    
                                    if msg["result"] is True and msg["error"] is None:
                                        accepts += 1
                                    else:
                                        rejects += 1
                                        print(f"Rejected Share! Error: {msg['error']}")
                                        
                        except json.JSONDecodeError:
                            pass # Not a valid JSON line
                except Exception:
                    pass

    print("=========================================")
    print("           PCAP STRATUM ANALYSIS         ")
    print("=========================================")
    print(f"1. Miner Connection Stability:")
    for ip, stats in connections.items():
        if stats["syn"] > 0 or stats["fin"] > 0 or stats["rst"] > 0:
            print(f"   - Miner IP {ip}: {stats['syn']} Connects, {stats['fin']} Disconnects (FIN), {stats['rst']} Resets (RST)")
            if stats["syn"] > 1 or stats["rst"] > 0:
                print(f"     [!] Warning: IP {ip} shows reconnects or connection resets.")
    
    print(f"\n2. Share Forwarding Efficiency:")
    print(f"   - Total Submits: {submits}")
    print(f"   - Total Accepts: {accepts}")
    print(f"   - Total Rejects: {rejects}")
    
    if latencies:
        avg_latency = sum(latencies) / len(latencies) * 1000
        max_latency = max(latencies) * 1000
        print(f"   - Avg Pool Response Latency: {avg_latency:.2f} ms")
        print(f"   - Max Pool Response Latency: {max_latency:.2f} ms")
        if avg_latency < 50:
            print("     [OK] Forwarding efficiency is Excellent (< 50ms)")
        elif avg_latency < 200:
            print("     [OK] Forwarding efficiency is Good")
        else:
            print("     [!] Forwarding efficiency is Slow (> 200ms)")
    else:
        print("   - Could not calculate latency (responses not matched to submits).")
        
    print(f"\n3. Fee Extraction (DevFee):")
    if fee_submits > 0:
        print(f"   - Identified {fee_submits} shares routed to Fee wallet.")
        print(f"   - DevFee mechanism is ACTIVE and working.")
    else:
        print("   - No explicit Fee shares found in this PCAP sample (might need longer capture).")
        
    print(f"\n4. Pool Rejections:")
    if submits > 0:
        reject_rate = (rejects / submits) * 100
        print(f"   - Rejection Rate: {reject_rate:.2f}%")
        if reject_rate > 1.0:
            print("     [!] Rejection rate is abnormally high!")
        else:
            print("     [OK] Rejection rate is healthy.")

if __name__ == '__main__':
    if len(sys.argv) > 1:
        analyze_pcap(sys.argv[1])
    else:
        print("Please provide pcap filepath.")
