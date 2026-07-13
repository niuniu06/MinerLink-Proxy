import sys

with open('frontend/src/components/Dashboard.vue', 'r', encoding='utf-8') as f:
    content = f.read()

target1 = "import axios from 'axios'"
content = content.replace(target1, "")

# fix evRes
target2 = "const evRes = await axios.get('/api/events')\n      events.value = evRes.data || []"
repl2 = "const evRes = await fetch('/api/events')\n      if (evRes.ok) {\n        events.value = await evRes.json() || []\n      }"
content = content.replace(target2, repl2)

# fix histRes
target3 = "const histRes = await axios.get(`/api/stats/history?port=`)\n        portHistoryData.value = histRes.data || []"
repl3 = "const histRes = await fetch(`/api/stats/history?port=`)\n        if (histRes.ok) {\n          portHistoryData.value = await histRes.json() || []\n        }"
content = content.replace(target3, repl3)

with open('frontend/src/components/Dashboard.vue', 'w', encoding='utf-8') as f:
    f.write(content)
