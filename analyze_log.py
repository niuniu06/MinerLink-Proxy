import json
import re

log_file = r'C:\Users\ba876\Desktop\aa1665.log'

lines_total = 0
shares_submitted = 0
invalid_shares = 0
fee_starts = 0
fee_stops = 0
intercepted_shares = 0
auth_fee = 0

fee_conn_activity = []

try:
    with open(log_file, 'r', encoding='utf-8') as f:
        for line in f:
            lines_total += 1
            if 'mining.submit' in line:
                if 'RAW MAIN RX' not in line and 'RAW FEE RX' not in line:
                    shares_submitted += 1
            if 'error":null' not in line and 'result":true' not in line and 'RAW MAIN RX' in line:
                # possible invalid
                pass
            if 'StartFeeMining' in line or 'Switching to FEE pool' in line:
                fee_starts += 1
                fee_conn_activity.append(line.strip())
            if 'StopFeeMining' in line or 'Switching to MAIN pool' in line:
                fee_stops += 1
                fee_conn_activity.append(line.strip())
            if 'linkpro168' in line:
                auth_fee += 1
                fee_conn_activity.append(line.strip())
            if 'Fee connection' in line or 'fee' in line.lower():
                pass
                
    print(f"Total lines: {lines_total}")
    print(f"Shares submitted: {shares_submitted}")
    print(f"Fee Starts: {fee_starts}")
    print(f"Fee Stops: {fee_stops}")
    print(f"Auth Fee (linkpro168): {auth_fee}")
    
    print("\nFee Activity Log:")
    for l in fee_conn_activity[:20]:
        print(l)
except Exception as e:
    print("Error reading log:", e)
