import struct
import socket
import sys

def search_pcap(filepath, target_ts, window=30):
    with open(filepath, 'rb') as f:
        global_header = f.read(24)
        if len(global_header) < 24:
            return
            
        magic_number = struct.unpack('<I', global_header[:4])[0]
        endian = '<' if magic_number == 0xa1b2c3d4 or magic_number == 0xa1b23c4d else '>'
        
        while True:
            header = f.read(16)
            if len(header) < 16:
                break
                
            ts_sec, ts_usec, incl_len, orig_len = struct.unpack(endian + 'IIII', header)
            packet_data = f.read(incl_len)
            
            if abs(ts_sec - target_ts) <= window:
                if b'{"' in packet_data:
                    try:
                        idx = packet_data.index(b'{"')
                        payload = packet_data[idx:].decode('utf-8', errors='ignore').strip()
                        print(f"[{ts_sec}] {payload}")
                    except:
                        pass

search_pcap(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 1782670262, 10)
