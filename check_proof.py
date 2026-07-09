import json
import base64
import gzip
import io

log_file = r'C:\Users\ba876\Desktop\aa1665.log'

with open(log_file, 'r', encoding='utf-8') as f:
    for line in f:
        if 'mining.submit' in line and 'plain_proof' in line:
            try:
                # Find the JSON part
                json_start = line.find('{')
                if json_start != -1:
                    json_str = line[json_start:]
                    data = json.loads(json_str)
                    if 'params' in data and 'plain_proof' in data['params']:
                        b64_data = data['params']['plain_proof']
                        compressed = base64.b64decode(b64_data)
                        with gzip.GzipFile(fileobj=io.BytesIO(compressed), mode='rb') as gz:
                            uncompressed = gz.read()
                        print("Found plain_proof length:", len(uncompressed))
                        # Try to decode it as string or hex
                        try:
                            print(uncompressed.decode('utf-8'))
                        except:
                            print(uncompressed.hex()[:500] + '...')
                        break
            except Exception as e:
                print("Error:", e)
