import re
import os
import json
from scapy.all import rdpcap, TCP, Raw

pcap_path = r'C:\Users\ba876\Desktop\etcproxy_capture.pcap'

def analyze_pcap():
    print("--- Detailed PCAP Analysis (TCP Payloads) ---")
    if not os.path.exists(pcap_path):
        print("PCAP not found.")
        return
        
    try:
        packets = rdpcap(pcap_path, count=100) # Read first 100 packets
        for pkt in packets:
            if pkt.haslayer(TCP) and pkt.haslayer(Raw):
                payload = pkt[Raw].load.decode('utf-8', errors='ignore').strip()
                if 'eth_submitWork' in payload or 'mining.submit' in payload or 'result' in payload or 'mining.notify' in payload:
                    # Print short version of payload to see flow
                    print(f"{pkt.time}: {payload[:200]}...")
    except Exception as e:
        print(f"Error parsing pcap: {e}")

if __name__ == '__main__':
    analyze_pcap()
