from scapy.all import rdpcap, TCP, IP
import sys

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    streams = {}
    
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            src = f"{pkt[IP].src}:{pkt[TCP].sport}"
            dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
            
            # Simple stream grouping by port pairs
            stream_key = tuple(sorted([src, dst]))
            if stream_key not in streams:
                streams[stream_key] = []
                
            payload = pkt[TCP].load
            try:
                decoded = payload.decode('utf-8', errors='ignore').strip()
                if decoded:
                    streams[stream_key].append(f"[{src} -> {dst}] {decoded}")
            except:
                pass

    for k, v in streams.items():
        print(f"--- Stream between {k[0]} and {k[1]} ---")
        for line in v:
            print(line.replace('\n', ' '))
        print("\n")
except Exception as e:
    print(f"Error: {e}")
