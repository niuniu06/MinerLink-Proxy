from scapy.all import rdpcap, TCP
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
fee_streams = set()

for p in packets:
    if TCP in p:
        payload = bytes(p[TCP].payload)
        if b'linkpro168' in payload:
            fee_streams.add((p[TCP].sport, p[TCP].dport))
            fee_streams.add((p[TCP].dport, p[TCP].sport))

for p in packets:
    if TCP in p:
        if (p[TCP].sport, p[TCP].dport) in fee_streams:
            payload = bytes(p[TCP].payload)
            if payload:
                try:
                    text = payload.decode('utf-8', 'ignore')
                    if text.strip():
                        print(f"[{p[TCP].sport} -> {p[TCP].dport}]: {text.strip()}")
                except:
                    pass
