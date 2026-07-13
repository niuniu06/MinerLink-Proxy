$content = Get-Content -Path "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\frontend\src\components\Dashboard.vue" -Raw

# 1. Add Layout blocks for HashrateChart and EventLog
$layoutInjection = @"
    <!-- Metrics Grid -->
    <div class="metrics-grid" :class="{ 'has-upgrade': updateInfo && updateInfo.hasUpdate }">
"@
$layoutReplacement = @"
    <!-- Chart and Events Layout -->
    <div class="dashboard-top-row">
      <div class="chart-section">
        <HashrateChart :historyData="statsHistory" title="全局算力曲线" />
      </div>
      <div class="events-section">
        <EventLog :events="eventLogs" />
      </div>
    </div>

    <!-- Metrics Grid -->
    <div class="metrics-grid" :class="{ 'has-upgrade': updateInfo && updateInfo.hasUpdate }">
"@
$content = $content -replace [regex]::Escape($layoutInjection), $layoutReplacement

# 2. Add Imports
$importInjection = @"
import CoinIcon from './CoinIcon.vue'
"@
$importReplacement = @"
import CoinIcon from './CoinIcon.vue'
import HashrateChart from './HashrateChart.vue'
import EventLog from './EventLog.vue'
import axios from 'axios'
"@
$content = $content -replace [regex]::Escape($importInjection), $importReplacement

# 3. Add Reactive variables for history and events
$varInjection = @"
  const upgradeStep = ref('正在联系服务器，准备下载...')
"@
$varReplacement = @"
  const upgradeStep = ref('正在联系服务器，准备下载...')
  
  const statsHistory = ref([])
  const eventLogs = ref([])
"@
$content = $content -replace [regex]::Escape($varInjection), $varReplacement

# 4. Add data fetching to fetchData and setinterval
$fetchInjection = @"
  const fetchGlobalStats = async () => {
    try {
      const res = await fetch('/api/global')
"@
$fetchReplacement = @"
  const fetchHistoryData = async () => {
    try {
      const [histRes, evRes] = await Promise.all([
        axios.get('/api/stats/history'),
        axios.get('/api/events')
      ])
      statsHistory.value = histRes.data || []
      eventLogs.value = evRes.data || []
    } catch (e) {
      console.error('Failed to load history or events:', e)
    }
  }

  const fetchGlobalStats = async () => {
    try {
      const res = await fetch('/api/global')
"@
$content = $content -replace [regex]::Escape($fetchInjection), $fetchReplacement

# 5. Call fetchHistoryData
$onMountedInjection = @"
  onMounted(() => {
    fetchGlobalStats()
    timer = setInterval(fetchGlobalStats, 2000)
"@
$onMountedReplacement = @"
  onMounted(() => {
    fetchGlobalStats()
    fetchHistoryData()
    timer = setInterval(() => {
        fetchGlobalStats()
        fetchHistoryData()
    }, 10000)
"@
$content = $content -replace [regex]::Escape($onMountedInjection), $onMountedReplacement

# 6. Add layout styles
$styleInjection = @"
<style scoped>
"@
$styleReplacement = @"
<style scoped>
.dashboard-top-row {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 2rem;
}
.chart-section {
  flex: 7;
  min-width: 0; /* Prevent flex overflow */
}
.events-section {
  flex: 3;
  min-width: 0;
}

@media (max-width: 1024px) {
  .dashboard-top-row {
    flex-direction: column;
  }
}
"@
$content = $content -replace [regex]::Escape($styleInjection), $styleReplacement

$content | Set-Content -Path "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\frontend\src\components\Dashboard.vue" -Encoding UTF8
