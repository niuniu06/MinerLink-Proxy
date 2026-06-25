import json
from scapy.all import rdpcap, TCP, IP, Raw
import time

def analyze_advanced(pcap_path):
    print("Loading PCAP for advanced analysis...")
    packets = rdpcap(pcap_path)
    
    total_bytes_up = 0
    total_bytes_down = 0
    tcp_retransmissions = 0
    proxy_turnaround_times = []
    
    # Track the last time a pool response was received to see how fast the proxy forwards it
    # We will match the payload content (e.g. {"id":1,"result":true,"error":null})
    pool_to_proxy = {} # payload -> time
    
    seen_seqs = set()

    for pkt in packets:
        if IP in pkt and TCP in pkt:
            src = pkt[IP].src
            dst = pkt[IP].dst
            sport = pkt[TCP].sport
            dport = pkt[TCP].dport
            seq = pkt[TCP].seq
            
            # Check for TCP retransmission (same sequence number sent from same source)
            seq_key = f"{src}:{sport}->{dst}:{dport}:{seq}"
            if seq_key in seen_seqs:
                tcp_retransmissions += 1
            seen_seqs.add(seq_key)
            
            if Raw in pkt:
                payload_len = len(pkt[Raw].load)
                try:
                    payload_str = pkt[Raw].load.decode('utf-8').strip()
                except:
                    payload_str = ""
                    
                if dport == 10690:
                    total_bytes_up += payload_len
                elif sport == 10690:
                    total_bytes_down += payload_len
                    # Proxy is sending to miner. Did we just receive this from the pool?
                    if payload_str in pool_to_proxy:
                        turnaround = (pkt.time - pool_to_proxy[payload_str]) * 1000000 # microseconds
                        if turnaround > 0 and turnaround < 100000: # filter out anomalies > 100ms
                            proxy_turnaround_times.append(turnaround)
                        del pool_to_proxy[payload_str] # clean up
                        
                elif sport == 2288 or sport == 1315:
                    # Pool is sending to proxy
                    pool_to_proxy[payload_str] = pkt.time

    avg_turnaround = sum(proxy_turnaround_times) / len(proxy_turnaround_times) if proxy_turnaround_times else 0
    
    print("\n==================================================")
    print(" *** ADVANCED NETWORK & EFFICIENCY REPORT ***")
    print("==================================================")
    print(f"Total Bytes Uploaded (Miner -> Proxy): {total_bytes_up / 1024:.2f} KB")
    print(f"Total Bytes Downloaded (Proxy -> Miner): {total_bytes_down / 1024:.2f} KB")
    print(f"TCP Retransmissions (Packet Loss Indicator): {tcp_retransmissions}")
    if tcp_retransmissions > 0:
        print(f"Network Health: Packet loss present ({tcp_retransmissions} dropped packets).")
    else:
        print("Network Health: Flawless (0 packet loss).")
        
    print(f"Proxy Internal Turnaround Time: {avg_turnaround:.2f} microseconds")
    if avg_turnaround < 1000:
        print("Code Efficiency RATING: S+ (Ultra-Low Latency, Sub-millisecond)")
    elif avg_turnaround < 5000:
        print("Code Efficiency RATING: A (Very Fast)")
    else:
        print("Code Efficiency RATING: B (Average)")

if __name__ == '__main__':
    analyze_advanced("C:\\Users\\ba876\\Desktop\\proxy_capture.pcap")
