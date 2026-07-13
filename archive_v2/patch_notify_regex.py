import codecs
import re

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'r', 'utf-8') as f:
    content = f.read()

def inject_cleanjobs(text):
    # Find the line: pendingDiff := s.PendingDiff
    # We want to wrap the following block with if isCleanJobs
    
    new_text = re.sub(
        r'(pendingDiff := s\.PendingDiff\s+if pendingDiff > 0 && pendingDiff != s\.LocalDiff \{)',
        r'''isCleanJobs := false
						if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
							if cj, ok := params[8].(bool); ok && cj {
								isCleanJobs = true
							}
						}

						// [Bugfix] Only flush pending difficulty if clean_jobs is true
						\1
							if !isCleanJobs {
								continue // Do not apply diff now, wait for a clean jobs notify
							}''',
        text
    )
    return new_text

content = inject_cleanjobs(content)

with codecs.open(r'C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\internal\proxy\session.go', 'w', 'utf-8') as f:
    f.write(content)

print("Regex patch applied successfully.")
