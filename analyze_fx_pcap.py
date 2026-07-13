import sys
from scapy.all import rdpcap, TCP, IP, Raw
import json
import re

pcap_file = r"C:\Users\ba876\Desktop\btcby_capture5.pcap"
print(f"Loading {pcap_file}...")
try:
    packets = rdpcap(pcap_file)
except Exception as e:
    print(f"Failed to load pcap: {e}")
    sys.exit(1)

connections = {} # key: (src_ip, src_port, dst_ip, dst_port)
miners = set()
pool_connections = set()
events = []

for pkt in packets:
    if IP in pkt and TCP in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        
        # Track connections/disconnections
        if pkt[TCP].flags.S: # SYN
            events.append({"type": "SYN", "time": float(pkt.time), "src": src, "dst": dst})
        if pkt[TCP].flags.R or pkt[TCP].flags.F: # RST or FIN
            events.append({"type": "FIN/RST", "time": float(pkt.time), "src": src, "dst": dst, "flags": str(pkt[TCP].flags)})
            
        if Raw in pkt:
            try:
                payload = pkt[Raw].load.decode('utf-8', errors='ignore')
                for line in payload.split('\n'):
                    line = line.strip()
                    if not line: continue
                    if '{' in line and '}' in line:
                        try:
                            # Extract json
                            json_str = line[line.find('{'):line.rfind('}')+1]
                            msg = json.loads(json_str)
                            
                            conn_type = "UNKNOWN"
                            if pkt[TCP].dport == 10510 or pkt[TCP].sport == 10510:
                                conn_type = "MINER<->FX"
                                if pkt[TCP].sport != 10510: miners.add(src)
                                else: miners.add(dst)
                            elif pkt[TCP].dport == 700 or pkt[TCP].sport == 700:
                                conn_type = "FX<->MAIN_POOL"
                                if pkt[TCP].dport == 700: pool_connections.add(src)
                            elif pkt[TCP].dport == 1301 or pkt[TCP].sport == 1301 or pkt[TCP].dport == 1315 or pkt[TCP].sport == 1315:
                                conn_type = "FX<->FEE_POOL"
                                if pkt[TCP].dport in [1301, 1315]: pool_connections.add(src)
                                
                            events.append({
                                "type": "JSON",
                                "time": float(pkt.time),
                                "src": src,
                                "dst": dst,
                                "conn": conn_type,
                                "msg": msg
                            })
                        except: pass
            except: pass

print(f"Total events extracted: {len(events)}")
print(f"Unique Miners found: {len(miners)}")
for m in miners: print(f"  - {m}")

# Print first 200 events to understand the structure
with open("fx_analysis_output.txt", "w", encoding="utf-8") as f:
    for e in events:
        if e['type'] == 'JSON':
            f.write(f"[{e['time']}] {e['conn']} | {e['src']} -> {e['dst']} | {json.dumps(e['msg'])}\n")
        else:
            f.write(f"[{e['time']}] {e['type']} | {e['src']} -> {e['dst']} | {e.get('flags', '')}\n")
print("Analysis saved to fx_analysis_output.txt")
