with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    code = f.read()

# Let's inspect the block handling mining.set_difficulty
import re
match = re.search(r'if method == "mining.set_difficulty"[\s\S]*?if isFirstDiff \{', code)
if match:
    print(match.group(0))
else:
    print("Not found")
