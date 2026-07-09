import sys
with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# The error is at lines 1677, 1679, 1727, 1729. These are inside readFeeLoop where it should be 'true'
for i in [1676, 1678, 1726, 1728]:
    lines[i] = lines[i].replace('isFee', 'true')

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.writelines(lines)
