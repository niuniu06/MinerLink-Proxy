from scapy.all import rdpcap
import json

packets = rdpcap(r'C:\Users\ba876\Desktop\prlproxy_capture.pcap')
for p in packets:
    try:
        payload = bytes(p.payload).decode('utf-8', 'ignore')
        if 'error' in payload and 'result' in payload:
            print(payload.split('{', 1)[1].split('}', 1)[0] + '}')
    except:
        pass
