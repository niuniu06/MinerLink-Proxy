from scapy.all import rdpcap, TCP, IP
import sys
from datetime import datetime

print("Loading pcap file... This might take a minute.")
pkts = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')
print(f"Loaded {len(pkts)} packets. Analyzing...")

first_time = None
proxy_ips = {'172.31.11.58', '172.31.12.58'}

# To track active sessions to avoid double counting FIN/RST
active_sessions = set()
# Miner IP -> list of disconnect times (in seconds from start)
disconnect_stats = {}

for p in pkts:
    if not (p.haslayer(TCP) and p.haslayer(IP)):
        continue
        
    if first_time is None:
        first_time = p.time
        
    src = p[IP].src
    dst = p[IP].dst
    sport = p[TCP].sport
    dport = p[TCP].dport
    
    # Identify if this packet is between miner and proxy
    is_miner_to_proxy = dst in proxy_ips
    is_proxy_to_miner = src in proxy_ips
    
    if not (is_miner_to_proxy or is_proxy_to_miner):
        continue
        
    miner_ip = src if is_miner_to_proxy else dst
    miner_port = sport if is_miner_to_proxy else dport
    
    # Session key
    session_key = (miner_ip, miner_port)
    
    flags = p[TCP].flags
    
    if flags & 0x02: # SYN
        active_sessions.add(session_key)
        
    elif (flags & 0x01) or (flags & 0x04): # FIN or RST
        if session_key in active_sessions:
            active_sessions.remove(session_key)
            if miner_ip not in disconnect_stats:
                disconnect_stats[miner_ip] = []
            rel_time = float(p.time - first_time)
            reason = "RST" if (flags & 0x04) else "FIN"
            disconnect_stats[miner_ip].append(f"{rel_time:.2f}s ({reason})")

print("\n--- FX Proxy Disconnect Analysis ---")
for ip, times in disconnect_stats.items():
    print(f"Miner {ip}: Kicked {len(times)} times")
    print(f"  Times: {', '.join(times)}")

