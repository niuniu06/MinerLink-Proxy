import pyshark
import json

cap = pyshark.FileCapture('C:/Users/ba876/Desktop/btc13_capture.pcap', display_filter='tcp.port == 10510 or tcp.port == 700')
for pkt in cap:
    try:
        if hasattr(pkt, 'tcp') and hasattr(pkt.tcp, 'payload'):
            payload = bytes.fromhex(pkt.tcp.payload.replace(':', '')).decode('utf-8')
            for line in payload.split('\n'):
                if 'mining.notify' in line:
                    data = json.loads(line)
                    if data.get('method') == 'mining.notify':
                        clean_jobs = data['params'][8]
                        src_port = pkt.tcp.srcport
                        dst_port = pkt.tcp.dstport
                        print(f"Time: {pkt.sniff_time} | Src: {src_port} -> Dst: {dst_port} | clean_jobs: {clean_jobs}")
    except Exception as e:
        pass
cap.close()
