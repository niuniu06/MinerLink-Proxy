from scapy.all import rdpcap, TCP, IP
import json

pcap_file = r'C:\Users\ba876\Desktop\prlproxy_capture.pcap'
try:
    packets = rdpcap(pcap_file)
    streams = {}
    
    for pkt in packets:
        if IP in pkt and TCP in pkt and hasattr(pkt[TCP], 'load'):
            src = f"{pkt[IP].src}:{pkt[TCP].sport}"
            dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
            
            stream_key = tuple(sorted([src, dst]))
            if stream_key not in streams:
                streams[stream_key] = []
                
            payload = pkt[TCP].load
            is_ascii = True
            try:
                decoded = payload.decode('ascii')
                for c in decoded:
                    if ord(c) < 32 and c not in ['\r', '\n', '\t']:
                        is_ascii = False
                        break
            except:
                is_ascii = False
                
            if is_ascii:
                streams[stream_key].append(f"[{src} -> {dst}] ASC: {payload.decode('ascii').strip()}")
            else:
                streams[stream_key].append(f"[{src} -> {dst}] HEX: {payload.hex()}")

    for k, v in streams.items():
        print(f"--- Stream {k[0]} <-> {k[1]} ---")
        for line in v[:20]:
            print(line)
        if len(v) > 20:
            print(f"... and {len(v)-20} more")
        print("\n")
except Exception as e:
    print(f"Error: {e}")
