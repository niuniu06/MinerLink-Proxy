import json
log_file = r'C:\Users\ba876\Desktop\aa1665.log'
shares = 0
try:
    with open(log_file, 'r', encoding='utf-8') as f:
        for line in f:
            if 'mining.submit' in line and 'RAW MAIN RX' not in line and 'RAW FEE RX' not in line:
                shares += 1
    print(f"Total shares in log: {shares}")
except Exception as e:
    print(e)
