from scapy.all import rdpcap, TCP, IP

cap = rdpcap('C:/Users/ba876/Desktop/v2.2.72proxy_capture.pcap')
for pkt in cap:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].sport == 10510:
        try:
            payload = bytes(pkt[TCP].payload).decode('utf-8', errors='ignore')
            if 'set_difficulty' in payload:
                print(f"[{pkt.time}] proxy -> miner: {payload.strip()}")
            elif 'mining.notify' in payload:
                # only print first notify to keep output short, wait I need to see if it immediately follows set_difficulty
                pass
        except Exception:
            pass
