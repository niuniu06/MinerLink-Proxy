target_hex = "0000000000000fffffffffffffffffffffffffffffffffffffffffffffffffff"
target_int = int(target_hex, 16)
diff = (2**256) / target_int
print(f"Eth Diff: {diff}")
