import sqlite3
import os
db_path = './data/proxy.db'
if not os.path.exists(db_path):
    print(f"DB not found: {db_path}")
else:
    c=sqlite3.connect(db_path)
    print([r[0] for r in c.execute('SELECT name FROM sqlite_master WHERE type="table"').fetchall()])
    print("Port Configs:")
    for row in c.execute('SELECT listen_port, algo, fee_percent, is_fee_enabled, dev_fee_enabled FROM port_configs').fetchall():
        print(row)
