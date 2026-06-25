import re
with open(r'C:\Users\ba876\Desktop\proxy_capture.pcap', 'rb') as f:
    data = f.read()
matches = re.findall(b'\{.*?\}', data)
for m in matches:
    try:
        s = m.decode('utf-8')
        if 'subscribe' in s or 'authorize' in s or 'error' in s:
            print(s)
    except:
        pass
