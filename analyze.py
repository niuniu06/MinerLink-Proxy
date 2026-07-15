from scapy.all import rdpcap, TCP, Raw
import collections

pkts = rdpcap('C:/Users/ba876/Desktop/btc11_capture.pcap')

streams = collections.defaultdict(list)
for pkt in pkts:
    if TCP in pkt:
        ip_layer = pkt.getlayer(1) # IP
        tcp_layer = pkt.getlayer(TCP)
        src = f"{ip_layer.src}:{tcp_layer.sport}"
        dst = f"{ip_layer.dst}:{tcp_layer.dport}"
        stream_id = tuple(sorted([src, dst]))
        
        flags = tcp_layer.flags
        payload = b""
        if Raw in pkt:
            payload = pkt[Raw].load
            
        streams[stream_id].append({
            'time': float(pkt.time),
            'src': src,
            'dst': dst,
            'flags': flags,
            'payload': payload
        })

print(f"Total streams: {len(streams)}")

for stream_id, packets in streams.items():
    # Check if stream ended with FIN or RST
    end_pkt = packets[-1]
    if 'F' in end_pkt['flags'] or 'R' in end_pkt['flags']:
        print(f"\n--- Stream {stream_id} ENDED (Flags: {end_pkt['flags']}) ---")
        # Print last few payloads
        payloads = [p for p in packets if p['payload']]
        for p in payloads[-3:]:
            prefix = "MINER -> PROXY" if ":10510" in p['dst'] else "PROXY -> MINER"
            print(f"[{p['time']:.2f}] {prefix}: {p['payload'][:150]}")
