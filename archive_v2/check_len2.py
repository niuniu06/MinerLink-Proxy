from scapy.all import rdpcap, TCP, IP

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            if pkt[IP].src.startswith('111.29.101.234'):
                try:
                    decoded = pkt[TCP].load.decode('ascii')
                except:
                    hex_val = pkt[TCP].load.hex()
                    print(f"Len {len(pkt[TCP].load)}: {hex_val}")
except Exception as e:
    print(f"Error: {e}")
