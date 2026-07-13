import re, sys
def extract_json(filepath):
    try:
        with open(filepath, 'rb') as f:
            data = f.read()
            # find strings that look like json (start with { and end with })
            matches = re.findall(b'\{[^{}]*"method"[^{}]*\}', data)
            # just extract ascii strings length > 20
            strings = re.findall(b'[ -~]{20,}', data)
            for s in strings:
                if b'mining.' in s or b'eth_' in s or b'id"' in s:
                    print(s.decode('ascii', errors='ignore'))
    except Exception as e:
        print(f"Error: {e}")

extract_json(sys.argv[1])
