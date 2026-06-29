import json
import re
import os
import struct

def find_last_json_messages(filepath, num_messages=20):
    with open(filepath, 'rb') as f:
        data = f.read()
    
    matches = list(re.finditer(b'\{.*?\}', data))
    
    if not matches:
        return "No JSON found"
        
    for m in matches[-num_messages:]:
        try:
            print(m.group(0).decode('utf-8', errors='ignore'))
        except:
            pass

find_last_json_messages(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 30)
