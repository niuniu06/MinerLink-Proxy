import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    for pkt in pcap:
        if pkt.haslayer(TCP) and pkt.haslayer(Raw):
            print(pkt[Raw].load)
            break
except Exception as e:
    print(f"Error: {e}")
