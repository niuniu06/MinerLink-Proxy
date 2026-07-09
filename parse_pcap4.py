from scapy.all import rdpcap, IP, TCP, Raw
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        if pkt[TCP].sport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if "result" in payload.lower():
                print(payload)
