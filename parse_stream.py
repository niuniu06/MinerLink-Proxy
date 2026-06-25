import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    
    # Find a stream where F2Pool dropped it
    target_stream = None
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(IP) and pkt[IP].src == '18.167.64.28' and (pkt[TCP].flags & 0x01 or pkt[TCP].flags & 0x04):
            target_stream = pkt[TCP].dport
            break

    print(f"Targeting port {target_stream} (Proxy port connecting to F2Pool)")
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(IP) and (pkt[TCP].sport == target_stream or pkt[TCP].dport == target_stream):
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
