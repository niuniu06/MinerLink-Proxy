from scapy.all import rdpcap, IP, TCP, Raw
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
submit_count = 0
result_true_count = 0

for pkt in packets:
    if IP in pkt and TCP in pkt and Raw in pkt:
        if pkt[TCP].dport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if 'mining.submit' in payload:
                submit_count += payload.count('mining.submit')
        elif pkt[TCP].sport == 5500:
            payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
            if 'result' in payload and 'true' in payload.lower():
                result_true_count += payload.count('result')

print(f"Submits sent to pool: {submit_count}")
print(f"Success replies (result:true) from pool: {result_true_count}")
