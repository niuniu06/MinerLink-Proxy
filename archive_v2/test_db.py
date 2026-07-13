import sqlite3
conn = sqlite3.connect(r"C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\proxy.db")
cursor = conn.cursor()
cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
print("Tables:", cursor.fetchall())
try:
    cursor.execute("SELECT port, coin_name, pool_address, fee_pool_address FROM ports")
    print("Ports:", cursor.fetchall())
except Exception as e:
    print(e)
conn.close()
