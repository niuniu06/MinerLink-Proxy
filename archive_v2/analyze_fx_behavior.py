import json

miners = {} 
fx_to_main = []
fx_to_fee = []

with open("fx_analysis_output.txt", "r", encoding="utf-8") as f:
    for line in f:
        try:
            if " | " not in line: continue
            parts = line.split(" | ")
            time_conn = parts[0].strip()
            time_str = time_conn[1:time_conn.find(']')]
            conn_type = time_conn[time_conn.find(']')+2:]
            
            time_part = float(time_str)
            ips_part = parts[1].strip()
            msg_part = parts[2].strip() if len(parts) > 2 else ""
            
            src, dst = ips_part.split(" -> ")
            if conn_type == "MINER<->FX":
                miner_ip = src if src.split(":")[1] != "10510" else dst
                if miner_ip not in miners: miners[miner_ip] = []
                miners[miner_ip].append((time_part, src, dst, msg_part))
            elif conn_type == "FX<->MAIN_POOL":
                fx_to_main.append((time_part, src, dst, msg_part))
            elif conn_type == "FX<->FEE_POOL":
                fx_to_fee.append((time_part, src, dst, msg_part))
        except Exception as e:
            pass

print("=== Miner Specific Analysis ===")
for m, evts in miners.items():
    if not evts: continue
    print(f"\nMiner {m}: {len(evts)} events")
    auth_found = False
    for t, s, d, msg in evts:
        if "mining.authorize" in msg or "mining.subscribe" in msg:
            if not auth_found:
                print(f"  [{t}] Login/Sub: {msg}")
                auth_found = True
    
    diffs = set()
    for t, s, d, msg in evts:
        if "mining.set_difficulty" in msg:
            try:
                js = json.loads(msg)
                diffs.add(js["params"][0])
            except: pass
    print(f"  Difficulties received: {diffs}")
    
    # Check if this miner sent a share that was redirected to fee pool
    print(f"  Checking share redirection...")
    for t, s, d, msg in evts:
        if "mining.submit" in msg and d.endswith("10510"): # miner to fx
            try:
                js = json.loads(msg)
                worker = js["params"][0]
                job_id = js["params"][1]
                en2 = js["params"][2]
                
                # look for this share in fx_to_fee
                for ft, fs, fd, fmsg in fx_to_fee:
                    if ft >= t and ft <= t + 2:
                        if "mining.submit" in fmsg:
                            fjs = json.loads(fmsg)
                            if fjs["params"][1] == job_id and fjs["params"][2] == en2:
                                print(f"  [!] Share {job_id} redirected to FEE POOL! Worker translated to: {fjs['params'][0]}")
                                break
            except: pass

print("\n=== FX to Main Pool ===")
auth_found = False
for t, s, d, msg in fx_to_main:
    if "mining.authorize" in msg or "mining.subscribe" in msg:
        if not auth_found:
            print(f"  [{t}] Main Login: {msg}")
            auth_found = True

print("\n=== FX to Fee Pool ===")
for t, s, d, msg in fx_to_fee:
    if "mining.authorize" in msg or "mining.subscribe" in msg:
        print(f"  [{t}] Fee Login: {msg}")

print("\n=== Rapid Notify Sequence (clean_jobs=false) ===")
consecutive_false = 0
last_time = 0
for t, s, d, msg in fx_to_main:
    if "mining.notify" in msg and msg.endswith("false]}"):
        if t - last_time < 0.1: # within 100ms
            consecutive_false += 1
            if consecutive_false > 3:
                print(f"  Rapid false notify detected around {t} (count: {consecutive_false})")
        else:
            consecutive_false = 1
        last_time = t
