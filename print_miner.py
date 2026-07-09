from scapy.all import rdpcap, TCP, IP

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            if pkt[IP].src.startswith('172.31.11.58') or pkt[IP].src.startswith('111.29.101.234'):
                try:
                    decoded = pkt[TCP].load.decode('ascii')
                    print(f"Miner Sent: {decoded.strip()}")
                except:
                    print(f"Miner Sent HEX: {pkt[TCP].load.hex()}")
except Exception as e:
    print(f"Error: {e}")
