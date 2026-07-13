import sys

with open('frontend/src/components/MinerChartModal.vue', 'r', encoding='utf-8') as f:
    content = f.read()

target1 = "import axios from 'axios'"
content = content.replace(target1, "")

target2 = "const res = await axios.get(/api/miner//history)\n    chartData.value = res.data || []"
repl2 = "const res = await fetch(/api/miner//history)\n    if (res.ok) {\n      chartData.value = await res.json() || []\n    }"
content = content.replace(target2, repl2)

with open('frontend/src/components/MinerChartModal.vue', 'w', encoding='utf-8') as f:
    f.write(content)
