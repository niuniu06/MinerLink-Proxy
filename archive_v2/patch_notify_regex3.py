import codecs
import re

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'r', 'utf-8') as f:
    content = f.read()

def inject_cleanjobs(text):
    new_text = re.sub(
        r'(\s*// Flush pending difficulty if any\s+pendingDiff := s\.PendingDiff\s+if pendingDiff > 0 && pendingDiff != s\.LocalDiff) (\{)',
        r'''
						isCleanJobs := false
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
							if cj, ok := params[8].(bool); ok && cj {
								isCleanJobs = true
							}
						}
\1 && isCleanJobs \2''',
        text
    )
    return new_text

content = inject_cleanjobs(content)

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'w', 'utf-8') as f:
    f.write(content)

print("Regex patch applied successfully.")
