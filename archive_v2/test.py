import time
import datetime

# 2026-06-29 21:00:00 BJT -> 13:00:00 UTC
dt = datetime.datetime(2026, 6, 29, 13, 0, 0, tzinfo=datetime.timezone.utc)
unix_sec = int(dt.timestamp())
unix_min = unix_sec // 60
print(f"At 21:00 BJT, unix minutes = {unix_min}, mod 100 = {unix_min % 100}")

dt2 = datetime.datetime(2026, 6, 29, 13, 34, 0, tzinfo=datetime.timezone.utc)
print(f"At 21:34 BJT, mod 100 = {(int(dt2.timestamp()) // 60) % 100}")

dt3 = datetime.datetime(2026, 6, 29, 14, 7, 0, tzinfo=datetime.timezone.utc)
print(f"At 22:07 BJT, mod 100 = {(int(dt3.timestamp()) // 60) % 100}")
