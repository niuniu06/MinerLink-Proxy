import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(IP) and (pkt[TCP].sport == 31208 or pkt[TCP].dport == 31208):
            if pkt.haslayer(Raw):
                payload = pkt[Raw].load.decode('utf-8', 'ignore').strip()
                if pkt[IP].src == '18.167.64.28':
                    print(f"[{pkt.time}] F2Pool -> Proxy: {payload}")
                else:
                    print(f"[{pkt.time}] Proxy -> F2Pool: {payload}")
            elif pkt[TCP].flags & 0x01:
                print(f"[{pkt.time}] FIN packet from {pkt[IP].src}")
except Exception as e:
    print(f"Error: {e}")
