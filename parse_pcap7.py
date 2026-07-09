from scapy.all import rdpcap, IP, TCP, Raw

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        if pkt[TCP].dport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if 'mining.submit' in payload:
                if 1783626020 < pkt.time < 1783626080:
                    print(f"[{pkt.time}] [{src} -> {dst}] {payload[:100]}")
