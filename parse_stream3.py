import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(IP) and (pkt[TCP].sport == 24399 or pkt[TCP].dport == 24399):
            if pkt.haslayer(Raw):
                payload = pkt[Raw].load.decode('utf-8', 'ignore').strip()
                if pkt[IP].src == '172.31.12.58':
                    print(f"[{pkt.time}] Proxy -> Miner: {payload}")
                else:
                    print(f"[{pkt.time}] Miner -> Proxy: {payload}")
            elif pkt[TCP].flags & 0x01:
                print(f"[{pkt.time}] FIN packet from {pkt[IP].src}")
except Exception as e:
    print(f"Error: {e}")
