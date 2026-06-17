import socket
import json
import time

try:
    s = socket.socket()
    s.settimeout(5)
    print("Connecting to etc.f2pool.com:8118...")
    s.connect(('etc.f2pool.com', 8118))
    
    login = {"id": 1, "method": "eth_submitLogin", "params": ["linkpro168", "dev"]}
    print(f"Sending: {json.dumps(login)}")
    s.send((json.dumps(login) + "\n").encode())
    
    time.sleep(1)
    resp = s.recv(1024)
    print(f"Received: {resp}")
    
    s.close()
except Exception as e:
    print(f"Error: {e}")
