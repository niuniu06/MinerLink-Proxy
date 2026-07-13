import sys

disconnects = []
reconnects = []

with open("fx_analysis_output.txt", "r", encoding="utf-8") as f:
    for line in f:
        if "FIN/RST" in line:
            disconnects.append(line.strip())
        elif "SYN" in line and "10510" in line:
            reconnects.append(line.strip())

print(f"Total Miner Reconnects (SYN): {len(reconnects)}")
print(f"Total Disconnects (FIN/RST): {len(disconnects)}")
for d in disconnects[:20]:
    print(d)

