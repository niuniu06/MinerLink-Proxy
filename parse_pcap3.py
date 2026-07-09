from scapy.all import rdpcap, IP, TCP, Raw
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        if pkt[TCP].dport == 5500 or pkt[TCP].sport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if "authorize" in payload.lower() or "login" in payload.lower() or "subscribe" in payload.lower():
                print(f"[{src} -> {dst}] {payload}")
