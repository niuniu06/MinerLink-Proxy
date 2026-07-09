import json
import re

def analyze(filepath):
    print(f"--- Analyzing {filepath} ---")
    with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
        lines = f.readlines()
        
    for idx, line in enumerate(lines):
        line = line.strip()
        
        # We look for ANY of these:
        if "switch" in line.lower() or "fee" in line.lower():
            if "FEE" not in line and "fee" not in line: 
                pass
            # Just print anything mentioning fee/switch that is not a giant JSON block
            if len(line) < 300:
                print(f"[{idx}] {line}")
                
        if "mining.set_difficulty" in line or "mining.set_extranonce" in line:
            print(f"[{idx}] {line[:200]}")
            
        if "reject" in line.lower():
             print(f"[{idx}] {line[:200]}")
             
        if "disconnect" in line.lower() or "reconnect" in line.lower():
            print(f"[{idx}] {line[:200]}")

analyze(r'C:\Users\ba876\Desktop\S21-14.log')
analyze(r'C:\Users\ba876\Desktop\S21-bb1.log')
