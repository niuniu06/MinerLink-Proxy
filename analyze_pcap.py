from scapy.all import rdpcap, TCP, IP, Raw
import sys

try:
    packets = rdpcap(r"C:\Users\ba876\Desktop\btcproxy_capture.pcap")
    disconnects = 0
    for pkt in packets:
        if TCP in pkt and IP in pkt:
            if pkt[TCP].flags.R or pkt[TCP].flags.F:
                print(f"[{pkt.time}] Disconnect (RST/FIN) {pkt[IP].src}:{pkt[TCP].sport} -> {pkt[IP].dst}:{pkt[TCP].dport}")
                disconnects += 1
            if Raw in pkt:
                payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
                if "mining.authorize" in payload or "mining.submit" in payload:
                    if len(payload) > 100:
                        payload = payload[:100] + "..."
                    #print(f"[{pkt.time}] {pkt[IP].src}:{pkt[TCP].sport} -> {pkt[IP].dst}:{pkt[TCP].dport} | {payload}")
    print(f"Total disconnects: {disconnects}")
except Exception as e:
    print(f"Error: {e}")
