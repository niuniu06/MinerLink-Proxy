from scapy.all import rdpcap, TCP, IP
import sys

print("Loading first 1000 packets to guess proxy IP...")
try:
    pkts = rdpcap(r'C:\Users\ba876\Desktop\fx-proxy_capture.pcap', count=1000)
    syn_dsts = {}
    for p in pkts:
        if p.haslayer(TCP) and p.haslayer(IP):
            if p[TCP].flags & 0x02: # SYN
                dst = p[IP].dst
                syn_dsts[dst] = syn_dsts.get(dst, 0) + 1
    print("SYN destinations:", syn_dsts)
except Exception as e:
    print(e)
