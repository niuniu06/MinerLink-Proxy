from scapy.all import rdpcap, TCP, IP

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            # Proxy is 172.31.12.58:11500, Miner is 111.29.101.234
            if pkt[IP].src == '172.31.12.58' and pkt[TCP].sport == 11500 and pkt[IP].dst == '111.29.101.234':
                try:
                    decoded = pkt[TCP].load.decode('ascii')
                    print(f"Proxy->Miner ASC: {decoded.strip()}")
                except:
                    print(f"Proxy->Miner HEX: {pkt[TCP].load.hex()}")
except Exception as e:
    print(f"Error: {e}")
