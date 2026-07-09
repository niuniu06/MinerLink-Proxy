import json
import sys
import re

def parse_log(file_path):
    print(f"--- Analysis for {file_path} ---")
    with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
        for line_num, line in enumerate(f, 1):
            line = line.strip()
            
            # Match raw rx/tx logs
            if "[RAW" in line or "RX]" in line or "TX]" in line:
                # find the json part
                json_match = re.search(r'(\{.*\})', line)
                if json_match:
                    try:
                        data = json.loads(json_match.group(1))
                        
                        # Print relevant events
                        if 'method' in data:
                            if data['method'] == 'mining.set_difficulty':
                                print(f"[{line_num}] DIFF SET: {data['params']}")
                            elif data['method'] == 'mining.set_extranonce':
                                print(f"[{line_num}] EXTRANONCE SET: {data['params']}")
                            elif data['method'] == 'mining.submit':
                                pass # ignore submits to avoid spam
                        
                        if 'error' in data and data['error'] is not None:
                            print(f"[{line_num}] SHARE REJECTED / ERROR: {data['error']} (id={data.get('id')})")
                    except:
                        pass
            
            # Print proxy internal events
            if "fee" in line.lower() and "switch" in line.lower():
                print(f"[{line_num}] FEE SWITCH: {line}")
            if "disconnect" in line.lower() or "reconnect" in line.lower():
                print(f"[{line_num}] CONNECTION EVENT: {line}")
            if "reject" in line.lower() and "rx]" not in line.lower():
                print(f"[{line_num}] REJECT EVENT: {line}")

parse_log(r'C:\Users\ba876\Desktop\S21-14.log')
parse_log(r'C:\Users\ba876\Desktop\S21-bb1.log')
