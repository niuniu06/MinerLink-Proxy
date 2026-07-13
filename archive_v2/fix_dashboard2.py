import sys

with open('frontend/src/components/Dashboard.vue', 'r', encoding='utf-8') as f:
    content = f.read()

target1 = "import axios from 'axios'\n"
content = content.replace(target1, "")

target2 = "const evRes = await axios.get('/api/events')\n    eventLogs.value = evRes.data || []"
repl2 = "const evRes = await fetch('/api/events')\n    if (evRes.ok) {\n      eventLogs.value = await evRes.json() || []\n    }"
content = content.replace(target2, repl2)

target3 = "const histRes = await axios.get(/api/stats/history?port=)\n      portHistories.value[expandedPort.value] = histRes.data || []"
repl3 = "const histRes = await fetch(/api/stats/history?port=)\n      if (histRes.ok) {\n        portHistories.value[expandedPort.value] = await histRes.json() || []\n      }"
content = content.replace(target3, repl3)

with open('frontend/src/components/Dashboard.vue', 'w', encoding='utf-8') as f:
    f.write(content)
