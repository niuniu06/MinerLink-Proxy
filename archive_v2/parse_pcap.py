from scapy.all import rdpcap, IP, TCP, Raw

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        if pkt[TCP].dport == 5500 or pkt[TCP].sport == 5500:
            if Raw in pkt:
                payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
                if payload:
                    print(f"[{src} -> {dst}] {payload}")
