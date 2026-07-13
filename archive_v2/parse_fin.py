import sys
try:
    from scapy.all import rdpcap, TCP, IP, Raw
    pcap = rdpcap('C:/Users/ba876/Desktop/proxy_capture.pcap')
    fin_senders = []
    for pkt in pcap:
        if pkt.haslayer(TCP) and (pkt[TCP].flags & 0x01 or pkt[TCP].flags & 0x04): # FIN or RST
            src = f"{pkt[IP].src}:{pkt[TCP].sport}"
            dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
            conn_id = f"{min(src,dst)}-{max(src,dst)}"
            if conn_id not in fin_senders:
                fin_senders.append(conn_id)
                print(f"[{pkt.time}] FIRST FIN/RST for {conn_id} from {src}")
except Exception as e:
    print(f"Error: {e}")
