from scapy.all import rdpcap, TCP
pkts = rdpcap('C:/Users/ba876/Desktop/btc11_capture.pcap')
for pkt in pkts:
    if TCP in pkt:
        ip_layer = pkt.getlayer(1)
        tcp_layer = pkt.getlayer(TCP)
        if 'R' in tcp_layer.flags:
            print(f"[{pkt.time:.2f}] RST from {ip_layer.src}:{tcp_layer.sport} to {ip_layer.dst}:{tcp_layer.dport}")
            break
