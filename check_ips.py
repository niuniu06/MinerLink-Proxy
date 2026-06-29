from scapy.all import rdpcap, TCP, IP
import sys
from collections import Counter

print("Loading pcap...")
pkts = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap')
proxy_ips = {'172.31.11.58', '172.31.12.58'}

syn_srcs = Counter()

for p in pkts:
    if p.haslayer(TCP) and p.haslayer(IP):
        if p[TCP].flags & 0x02: # SYN
            if p[IP].dst in proxy_ips:
                syn_srcs[p[IP].src] += 1

print("Top IPs connecting TO proxy:")
for ip, count in syn_srcs.most_common(20):
    print(f"{ip}: {count} connections")
