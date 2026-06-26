from scapy.all import rdpcap, TCP, IP
import sys
import os

pcap_file = r'C:\Users\ba876\Desktop\proxy_capture.pcap'
if not os.path.exists(pcap_file):
    print('File not found')
    sys.exit(1)

print('Reading pcap...')
packets = rdpcap(pcap_file)
print(f'Total packets: {len(packets)}')

streams = {}

for pkt in packets:
    if IP in pkt and TCP in pkt:
        src = f"{pkt[IP].src}:{pkt[TCP].sport}"
        dst = f"{pkt[IP].dst}:{pkt[TCP].dport}"
        
        # Sort so client-server is same stream
        if pkt[TCP].sport > pkt[TCP].dport:
            stream_id = f"{src}-{dst}"
            is_client = True
        else:
            stream_id = f"{dst}-{src}"
            is_client = False
            
        if stream_id not in streams:
            streams[stream_id] = {'start_time': pkt.time, 'end_time': None, 'closed_by': None, 'payloads': [], 'is_reset': False}
            
        if pkt[TCP].flags & 0x01: # FIN
            if streams[stream_id]['closed_by'] is None:
                streams[stream_id]['closed_by'] = 'Client' if is_client else 'Server'
                streams[stream_id]['end_time'] = pkt.time
                
        if pkt[TCP].flags & 0x04: # RST
            if streams[stream_id]['closed_by'] is None:
                streams[stream_id]['closed_by'] = ('Client' if is_client else 'Server') + ' (RST)'
                streams[stream_id]['end_time'] = pkt.time
                streams[stream_id]['is_reset'] = True
                
        payload = bytes(pkt[TCP].payload)
        if len(payload) > 0:
            streams[stream_id]['payloads'].append((pkt.time, 'Client' if is_client else 'Server', payload))

for sid, data in streams.items():
    if data['closed_by'] is not None:
        duration = data['end_time'] - data['start_time']
        print(f"\nStream {sid} closed by {data['closed_by']} after {float(duration):.2f}s")
        for time, sender, payload in data['payloads']:
            try:
                print(f"  [{float(time - data['start_time']):.2f}s] {sender}: {payload.decode('utf-8').strip()}")
            except:
                print(f"  [{float(time - data['start_time']):.2f}s] {sender}: <binary data>")
