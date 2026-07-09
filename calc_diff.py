import math

target_hex = "0000000000000fffffffffffffffffffffffffffffffffffffffffffffffffff"
target_int = int(target_hex, 16)
max_target = int("00000000ffff0000000000000000000000000000000000000000000000000000", 16)

diff = max_target / target_int
print(f"Difficulty: {diff}")
