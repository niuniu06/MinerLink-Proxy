from scapy.all import rdpcap, IP, TCP, Raw

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        if pkt[TCP].dport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if 'mining.submit' in payload:
                print(f"[{pkt.time}]")
