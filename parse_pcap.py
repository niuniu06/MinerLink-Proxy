import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(Raw):
            payload = pkt[Raw].load.decode('utf-8', 'ignore')
            if 'mining.set_difficulty' in payload:
                print(f"[{pkt.time}] Found set_diff: {payload.strip()}")
        if pkt.haslayer(TCP) and (pkt[TCP].flags & 0x01 or pkt[TCP].flags & 0x04): # FIN or RST
            print(f"[{pkt.time}] FIN/RST src={pkt[IP].src}:{pkt[TCP].sport} dst={pkt[IP].dst}:{pkt[TCP].dport}")
except Exception as e:
    print(f"Error: {e}")
