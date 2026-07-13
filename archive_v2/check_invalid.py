import json

log_file = r'C:\Users\ba876\Desktop\aa1665.log'

try:
    with open(log_file, 'r', encoding='utf-8') as f:
        for i, line in enumerate(f):
            if 'result":false' in line.replace(' ', '') or '"error":[' in line.replace(' ', ''):
                print(f"Line {i+1}: {line.strip()}")
                
            # Also look for any pool response that has an error string
            if 'RAW MAIN RX' in line or 'RAW FEE RX' in line:
                try:
                    json_start = line.find('{')
                    if json_start != -1:
                        data = json.loads(line[json_start:])
                        if data.get('error') is not None:
                            print(f"Error found on Line {i+1}: {line.strip()}")
                        elif data.get('result') == False:
                            print(f"Result false found on Line {i+1}: {line.strip()}")
                except:
                    pass
