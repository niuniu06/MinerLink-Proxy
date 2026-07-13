import sqlite3
conn = sqlite3.connect(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\data.db')
cursor = conn.cursor()
cursor.execute('SELECT listen_port, dev_fee_percent, operator_fee_percent FROM proxy_configs')
for row in cursor.fetchall():
    print(row)
