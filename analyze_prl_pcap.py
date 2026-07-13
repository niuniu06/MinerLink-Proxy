import sys
from scapy.all import rdpcap, TCP, IP

pcap_file = r"C:\Users\ba876\Desktop\prl_capture.pcap"
print(f"Loading {pcap_file}...")
try:
    packets = rdpcap(pcap_file)
except Exception as e:
    print(f"Failed: {e}")
    sys.exit(1)

events = []
for pkt in packets:
    if IP in pkt and TCP in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        if pkt[TCP].flags.S:
            events.append(f"[{pkt.time}] SYN | {src} -> {dst}")
        if pkt[TCP].flags.R or pkt[TCP].flags.F:
            events.append(f"[{pkt.time}] FIN/RST ({pkt[TCP].flags}) | {src} -> {dst}")

with open("prl_disconnects.txt", "w", encoding="utf-8") as f:
    for e in events:
        f.write(e + "\n")
print(f"Total events: {len(events)}")
