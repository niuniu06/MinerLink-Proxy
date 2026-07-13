from scapy.all import rdpcap, TCP, IP

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            try:
                decoded = pkt[TCP].load.decode('ascii')
                if 'set_' in decoded:
                    print(decoded.strip())
            except:
                pass
except Exception as e:
    print(f"Error: {e}")
