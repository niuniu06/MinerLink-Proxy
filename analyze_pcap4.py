from scapy.all import rdpcap, TCP, IP

cap = rdpcap('C:/Users/ba876/Desktop/v2.2.72proxy_capture.pcap')
for pkt in cap:
    if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[TCP].sport == 10510:
        try:
            payload = bytes(pkt[TCP].payload).decode('utf-8', errors='ignore')
            if '1783362033' in str(pkt.time) or '1783362051' in str(pkt.time):
                print(f"[{pkt.time}] proxy -> miner: {payload}")
        except Exception:
            pass
