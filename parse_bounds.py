import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    print(f"Total packets: {len(pcap)}")
    if len(pcap) > 0:
        print(f"Earliest time: {pcap[0].time}")
        print(f"Latest time: {pcap[-1].time}")
except Exception as e:
    print(f"Error: {e}")
