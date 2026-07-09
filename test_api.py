import urllib.request
import json

try:
    req = urllib.request.Request('http://127.0.0.1:10000/api/miners?port=10715&page=1&limit=10')
    with urllib.request.urlopen(req) as response:
        miners = json.loads(response.read().decode())
        print("Miners:", miners)
        
        if miners and miners['data'] and len(miners['data']) > 0:
            miner_id = miners['data'][0]['id']
            print("Fetching history for miner:", miner_id)
            
            req2 = urllib.request.Request(f'http://127.0.0.1:10000/api/miner/{miner_id}/history')
            with urllib.request.urlopen(req2) as response2:
                history = json.loads(response2.read().decode())
                print(f"History length: {len(history)}")
                if len(history) > 0:
                    print("First item:", history[0])
                else:
                    print("History is EMPTY!")
except Exception as e:
    print("Error:", e)
