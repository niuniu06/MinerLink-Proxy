import re
with open('C:/Users/ba876/Desktop/btc13_capture.pcap', 'rb') as f:
    data = f.read()

import string
printable = bytes(string.printable, 'ascii')
res = []
current = bytearray()
for b in data:
    if b in printable:
        current.append(b)
    else:
        if b'mining.notify' in current:
            res.append(current.decode('ascii'))
        current = bytearray()

true_count = sum(1 for s in res if 'true]}' in s)
false_count = sum(1 for s in res if 'false]}' in s)
print(f"True count: {true_count}")
print(f"False count: {false_count}")
