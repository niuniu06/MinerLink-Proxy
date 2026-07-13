from scapy.all import rdpcap, IP, TCP, Raw

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        if pkt[TCP].sport == 49106 or pkt[TCP].dport == 49106:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            print(f"[{pkt.time}] [{src} -> {dst}] {payload[:100]}")
