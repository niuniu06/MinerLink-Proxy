import re

with open('C:/Users/ba876/Desktop/btc13_capture.pcap', 'rb') as f:
    data = f.read()

# find all ascii strings containing mining.notify
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

for s in set(res[:30]):
    print(s)
