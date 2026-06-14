<template>
  <div class="dashboard-container">
    <!-- Top System Metrics Cards (Glassmorphism) -->
    <div class="metrics-grid" :class="{ 'has-upgrade': updateInfo && updateInfo.hasUpdate }">
      <!-- New Version Card (If hasUpdate is true) -->
      <div class="metric-card upgrade-card" v-if="updateInfo && updateInfo.hasUpdate">
        <div class="upgrade-glow"></div>
        <div class="metric-info">
          <h3>发现新版本 ({{ updateInfo.latestVersion }})</h3>
          <div class="upgrade-action-row">
            <button class="btn-upgrade-now" @click="upgradeSystem">一键热升级</button>
            <span class="sub-label" :title="updateInfo.changelog">包含新优化及Bug修复</span>
          </div>
        </div>
        <div class="metric-visual">
          <div class="arrow-up-glow">⇧</div>
        </div>
      </div>

      <!-- CPU Usage Card -->
      <div class="metric-card sys-card">
        <div class="card-glow"></div>
        <div class="metric-info">
          <h3>服务器 CPU</h3>
          <div class="metric-main">
            <span class="big-val">{{ sysStatus ? sysStatus.cpuPercent : 0 }}%</span>
            <span class="sub-label">核心处理器负载</span>
          </div>
        </div>
        <div class="metric-visual">
          <svg class="progress-ring" width="60" height="60">
            <circle class="ring-bg" cx="30" cy="30" r="24" />
            <circle 
              class="ring-bar cpu-bar" 
              cx="30" 
              cy="30" 
              r="24" 
              :style="{ strokeDashoffset: calculateOffset(sysStatus ? sysStatus.cpuPercent : 0) }" 
            />
          </svg>
        </div>
      </div>

      <!-- Memory Usage Card -->
      <div class="metric-card sys-card">
        <div class="card-glow"></div>
        <div class="metric-info">
          <h3>服务器内存</h3>
          <div class="metric-main">
            <span class="big-val">{{ sysStatus ? sysStatus.memoryPercent : 0 }}%</span>
            <span class="sub-label">物理内存占用</span>
          </div>
        </div>
        <div class="metric-visual">
          <svg class="progress-ring" width="60" height="60">
            <circle class="ring-bg" cx="30" cy="30" r="24" />
            <circle 
              class="ring-bar mem-bar" 
              cx="30" 
              cy="30" 
              r="24" 
              :style="{ strokeDashoffset: calculateOffset(sysStatus ? sysStatus.memoryPercent : 0) }" 
            />
          </svg>
        </div>
      </div>

      <!-- Active Miners Card -->
      <div class="metric-card">
        <div class="metric-info">
          <h3>在线矿机总数</h3>
          <div class="metric-main">
            <span class="big-val">{{ totalActiveMiners }}</span>
            <span class="sub-label">集群连接机器台数</span>
          </div>
        </div>
        <div class="metric-icon">💻</div>
      </div>

      <!-- Uptime Card -->
      <div class="metric-card">
        <div class="metric-info">
          <h3>运行时长</h3>
          <div class="metric-main">
            <span class="big-val uptime-val" style="font-size: 1.8rem;">{{ formatUptime(sysStatus ? sysStatus.uptimeSeconds : 0) }}</span>
            <span class="sub-label">程序持续运行时间</span>
          </div>
        </div>
        <div class="metric-icon">🏃</div>
      </div>
    </div>

    <!-- Main Active Ports Panel -->
    <div class="ports-panel">
      <div class="panel-header">
        <h2>代理端口总览 <span class="badge">{{ configs.length }} 个活动端口</span></h2>
      </div>

      <div v-if="loading" class="loading-state">加载数据中...</div>
      <div v-else-if="configs.length === 0" class="empty-state">
        暂无运行中的中转端口，点击右下角按钮添加端口配置
      </div>

      <div v-else class="table-container">
        <table class="ports-table">
          <thead>
            <tr>
              <th width="30"></th>
              <th width="80">币种</th>
              <th width="90">本地端口</th>
              <th>目标矿池</th>
              <th width="90">在线矿机</th>
              <th width="150">抽水设置</th>
              <th width="160">提交 / 拦截份额</th>
              <th>运行状态</th>
              <th width="240" style="text-align: right">操作</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="cfg in configs" :key="cfg.listenPort">
              <!-- Main Port Row -->
              <tr 
                class="port-row" 
                :class="{ active: expandedPort === cfg.listenPort }"
                @click="toggleExpand(cfg.listenPort)"
              >
                <td class="td-expand">
                  <span class="arrow-icon" :class="{ open: expandedPort === cfg.listenPort }">▸</span>
                </td>
                <td class="coin-cell">
                  <span class="coin-logo">{{ cfg.coinName || 'ETH' }}</span>
                </td>
                <td class="port-cell">
                  <code>{{ cfg.listenPort }}</code>
                </td>
                <td class="pool-cell" :title="cfg.PoolAddress">
                  <span class="pool-addr">{{ cfg.PoolAddress }}</span>
                </td>
                <td class="miners-cell">
                  <span class="miner-badge" :class="{ 'has-miners': getStatsForPort(cfg.listenPort).activeMiners > 0 }">
                    {{ getStatsForPort(cfg.listenPort).activeMiners }} 台
                  </span>
                </td>
                <td class="fee-cell">
                  <span class="fee-badge">
                    作者 {{ cfg.devFeePercent }}%
                    <span v-if="cfg.operatorFeePercent > 0"> + 运营 {{ cfg.operatorFeePercent }}%</span>
                  </span>
                </td>
                <td class="shares-cell">
                  <span class="sh-total">{{ getStatsForPort(cfg.listenPort).totalShares }}</span> /
                  <span class="sh-fee">{{ getStatsForPort(cfg.listenPort).totalFeeShares }}</span>
                </td>
                <td class="status-cell">
                  <div class="status-indicator-row" :class="{ smooth: cfg.enableSmoothFee }">
                    <span class="pulse-dot"></span>
                    {{ cfg.enableSmoothFee ? '平滑无感中' : '静默拦截中' }}
                  </div>
                </td>
                <td class="actions-cell" style="text-align: right" @click.stop>
                  <button class="tbl-btn" @click="toggleExpand(cfg.listenPort)">
                    {{ expandedPort === cfg.listenPort ? '收起矿机' : '管理矿机' }}
                  </button>
                  <button class="tbl-btn btn-edit" @click="$emit('edit-config', cfg)">编辑</button>
                  <button class="tbl-btn btn-del" @click="deletePort(cfg.listenPort)">删除</button>
                </td>
              </tr>

              <!-- Sub-row containing MinerTable -->
              <tr v-if="expandedPort === cfg.listenPort" class="miner-detail-row">
                <td colspan="9" class="detail-container">
                  <div class="detail-card">
                    <div class="detail-header">
                      <h4>端口 {{ cfg.listenPort }} 在线矿机实时监控</h4>
                    </div>
                    <MinerTable :port="cfg.listenPort" />
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Upgrade Progress Overlay -->
    <div class="upgrade-overlay" v-if="upgrading">
      <div class="upgrade-modal">
        <div class="upgrade-spinner"></div>
        <h3>系统自动热升级中</h3>
        <p class="upgrade-step">{{ upgradeStep }}</p>
        <p class="upgrade-tips">升级下载通常需要 10-30 秒，期间端口服务可能短暂中断，请勿关闭系统电源。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import MinerTable from './MinerTable.vue'

const props = defineProps({
  sysStatus: Object,
  updateInfo: Object
})

const upgrading = ref(false)
const upgradeStep = ref('正在联系服务器，准备下载...')

const configs = ref([])
const stats = ref([])
const loading = ref(true)
const expandedPort = ref(null)
let intervalId = null

const fetchConfig = async () => {
  try {
    const res = await fetch('/api/config')
    if (res.ok) configs.value = await res.json()
  } catch (e) {
    console.error(e)
  }
}

const fetchStats = async () => {
  try {
    const res = await fetch('/api/stats')
    if (res.ok) stats.value = await res.json()
  } catch (e) {
    console.error(e)
  }
}

const performUpgrade = () => {
  // Placeholder logic handled globally
};

const formatUptime = (seconds) => {
  if (!seconds) return '0分';
  const d = Math.floor(seconds / (3600 * 24));
  const h = Math.floor((seconds % (3600 * 24)) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}天${h}时${m}分`;
  if (h > 0) return `${h}时${m}分`;
  return `${m}分`;
}; 

const upgradeSystem = async () => {
  if (!confirm(`确定要将系统从 ${props.updateInfo.currentVersion} 升级至 ${props.updateInfo.latestVersion} 吗？\n升级过程中代理端口将短暂重启，矿机会自动重新连接。`)) return
  
  upgrading.value = true
  upgradeStep.value = '正在下载新版可执行文件，请稍候...'
  
  try {
    const urlParams = new URLSearchParams(window.location.search);
    const mockParam = urlParams.get('mock') ? '?mock=1' : '';
    const res = await fetch('/api/system/upgrade' + mockParam, { method: 'POST' })
    if (res.ok) {
      upgradeStep.value = '新版可执行文件下载完成，正在热替换并重启服务...'
      setTimeout(startReconnecting, 3000)
    } else {
      const data = await res.json()
      alert(data.error || '升级失败')
      upgrading.value = false
    }
  } catch (e) {
    console.error(e)
    alert('连接升级 API 失败')
    upgrading.value = false
  }
}

const startReconnecting = () => {
  let attempts = 0
  const timer = setInterval(async () => {
    attempts++
    upgradeStep.value = `正在重新连接后台服务... (尝试 ${attempts} 次)`
    
    try {
      const res = await fetch('/api/system/status')
      if (res.ok) {
        const status = await res.json()
        if (status && status.cpuPercent !== undefined) {
          clearInterval(timer)
          upgradeStep.value = '连接成功，正在刷新页面...'
          setTimeout(() => {
            location.reload()
          }, 1000)
        }
      }
    } catch (e) {
      // Ignore connection failures during restart
    }
    
    if (attempts > 30) {
      clearInterval(timer)
      alert('重启超时，请手动刷新页面或登录服务器检查程序状态！')
      upgrading.value = false
    }
  }, 2000)
}

defineExpose({ refresh })

const getStatsForPort = (port) => {
  return stats.value.find(s => s.listenPort === port) || {
    activeMiners: 0,
    totalShares: 0,
    totalFeeShares: 0
  }
}

const toggleExpand = (port) => {
  if (expandedPort.value === port) {
    expandedPort.value = null
  } else {
    expandedPort.value = port
  }
}

const deletePort = async (port) => {
  if (!confirm(`确定要彻底删除端口 ${port} 的代理配置吗？此操作会导致当前连接 of 矿机断开中转。`)) return
  try {
    const res = await fetch('/api/config/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ listenPort: port })
    })
    if (res.ok) {
      refresh()
      if (expandedPort.value === port) expandedPort.value = null
    } else {
      const data = await res.json()
      alert(data.error || '删除失败')
    }
  } catch (e) {
    console.error(e)
  }
}

// Global Metrics computation
const totalActiveMiners = computed(() => {
  return stats.value.reduce((acc, curr) => acc + (curr.activeMiners || 0), 0)
})

const totalShares = computed(() => {
  return stats.value.reduce((acc, curr) => acc + (curr.totalShares || 0), 0)
})

const totalFeeShares = computed(() => {
  return stats.value.reduce((acc, curr) => acc + (curr.totalFeeShares || 0), 0)
})

const interceptRate = computed(() => {
  if (totalShares.value === 0) return '0.0'
  return ((totalFeeShares.value * 100) / totalShares.value).toFixed(1)
})

// Circular progress offset (radius = 24, circumference = 150.796)
const calculateOffset = (percent) => {
  const c = 150.796
  const p = Math.min(100, Math.max(0, percent))
  return c - (p / 100) * c
}

onMounted(async () => {
  await fetchConfig()
  await fetchStats()
  loading.value = false
  intervalId = setInterval(fetchStats, 2000)
})

onUnmounted(() => {
  clearInterval(intervalId)
})
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 1.8rem;
}

/* System Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
}

.metric-card {
  position: relative;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  transition: all 0.3s ease;
}

.metric-card:hover {
  transform: translateY(-2px);
  border-color: var(--accent-blue);
  box-shadow: 0 6px 25px rgba(88, 166, 255, 0.15);
}

.sys-card {
  background: rgba(22, 27, 34, 0.7);
  backdrop-filter: blur(10px);
}

.card-glow {
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(88, 166, 255, 0.04) 0%, transparent 60%);
  pointer-events: none;
  animation: rotateGlow 15s linear infinite;
}

@keyframes rotateGlow {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.metric-info h3 {
  margin: 0 0 0.5rem 0;
  font-size: 0.85rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.metric-main {
  display: flex;
  flex-direction: column;
}

.big-val {
  font-size: 1.8rem;
  font-weight: 700;
  color: var(--text-main);
  font-family: 'SF Mono', Consolas, monospace;
}

.total-fee {
  color: var(--accent-blue);
  text-shadow: 0 0 10px rgba(88, 166, 255, 0.3);
}

.sub-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.2rem;
}

.metric-visual {
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Circular Progress Style */
.progress-ring {
  transform: rotate(-90deg);
}

.ring-bg {
  fill: none;
  stroke: rgba(255, 255, 255, 0.05);
  stroke-width: 4.5;
}

.ring-bar {
  fill: none;
  stroke-width: 4.5;
  stroke-linecap: round;
  stroke-dasharray: 150.796;
  stroke-dashoffset: 150.796;
  transition: stroke-dashoffset 0.6s ease;
}

.cpu-bar {
  stroke: var(--accent-blue);
  filter: drop-shadow(0 0 3px rgba(88, 166, 255, 0.5));
}

.mem-bar {
  stroke: var(--accent-cyan);
  filter: drop-shadow(0 0 3px rgba(88, 166, 255, 0.5));
}

.metric-icon {
  font-size: 2rem;
  opacity: 0.8;
  filter: drop-shadow(0 0 5px rgba(255, 255, 255, 0.1));
}
.fee-icon {
  color: var(--accent-blue);
  filter: drop-shadow(0 0 8px rgba(88, 166, 255, 0.3));
}

/* Active Ports Panel */
.ports-panel {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.panel-header {
  margin-bottom: 1.25rem;
}

.panel-header h2 {
  margin: 0;
  font-size: 1.2rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.8rem;
}

.badge {
  background: rgba(88, 166, 255, 0.1);
  color: var(--accent-blue);
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 20px;
  border: 1px solid rgba(88, 166, 255, 0.2);
}

/* Table Design */
.table-container {
  overflow-x: auto;
}

.ports-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.ports-table th {
  color: var(--text-muted);
  font-size: 0.8rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 1rem 0.8rem;
  border-bottom: 1px solid var(--card-border);
}

.ports-table td {
  padding: 1rem 0.8rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
  font-size: 0.88rem;
}

/* Rows & Hover Animations */
.port-row {
  cursor: pointer;
  transition: all 0.2s ease;
}

.port-row:hover {
  background: rgba(255, 255, 255, 0.02);
}

.port-row.active {
  background: rgba(88, 166, 255, 0.03);
}

.td-expand {
  text-align: center;
  user-select: none;
}

.arrow-icon {
  display: inline-block;
  color: var(--text-muted);
  transition: transform 0.2s ease;
}

.arrow-icon.open {
  transform: rotate(90deg);
  color: var(--accent-blue);
}

.coin-logo {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--card-border);
  color: var(--text-main);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: bold;
}

.port-cell code {
  background: rgba(88, 166, 255, 0.08);
  border: 1px solid rgba(88, 166, 255, 0.15);
  color: var(--accent-blue);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'SF Mono', Consolas, monospace;
}

.pool-cell {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pool-addr {
  color: var(--text-muted);
}

.miner-badge {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--card-border);
  color: var(--text-muted);
  padding: 2px 8px;
  border-radius: 20px;
  font-size: 0.78rem;
}

.miner-badge.has-miners {
  background: rgba(63, 185, 80, 0.1);
  border-color: rgba(63, 185, 80, 0.2);
  color: var(--accent-green);
  font-weight: 600;
}

.fee-badge {
  font-size: 0.8rem;
  color: var(--accent-cyan);
}

.sh-total {
  font-family: monospace;
}
.sh-fee {
  font-family: monospace;
  color: var(--accent-blue);
  font-weight: 600;
}

/* Status Indicator style */
.status-indicator-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent-green);
  background: rgba(63, 185, 80, 0.08);
  padding: 3px 8px;
  border-radius: 4px;
  border: 1px solid rgba(63, 185, 80, 0.15);
}

.status-indicator-row.smooth {
  color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.08);
  border-color: rgba(88, 166, 255, 0.15);
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  animation: pulseLight 1.8s infinite;
}

@keyframes pulseLight {
  0% { box-shadow: 0 0 0 0 rgba(63, 185, 80, 0.4); }
  70% { box-shadow: 0 0 0 6px rgba(63, 185, 80, 0); }
  100% { box-shadow: 0 0 0 0 rgba(63, 185, 80, 0); }
}

.status-indicator-row.smooth .pulse-dot {
  animation-name: pulseSmooth;
}
@keyframes pulseSmooth {
  0% { box-shadow: 0 0 0 0 rgba(88, 166, 255, 0.4); }
  70% { box-shadow: 0 0 0 6px rgba(88, 166, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(88, 166, 255, 0); }
}

/* Table buttons */
.tbl-btn {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--card-border);
  color: var(--text-main);
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 0.78rem;
  cursor: pointer;
  margin-left: 6px;
  font-weight: 500;
  transition: all 0.2s;
}

.tbl-btn:hover {
  background: rgba(88, 166, 255, 0.08);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.btn-edit:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

.btn-del:hover {
  background: rgba(248, 81, 73, 0.08);
  border-color: var(--accent-red);
  color: var(--accent-red);
}

/* Detailed expanded Row */
.miner-detail-row td {
  padding: 0;
  border-bottom: 1px solid var(--card-border);
  background: rgba(13, 17, 23, 0.4);
}

.detail-container {
  padding: 1rem 1.5rem !important;
}

.detail-card {
  background: rgba(22, 27, 34, 0.3);
  border: 1px solid rgba(88, 166, 255, 0.1);
  border-radius: 8px;
  padding: 1.25rem;
  box-shadow: inset 0 2px 10px rgba(0, 0, 0, 0.2);
}

.detail-header {
  margin-bottom: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  padding-bottom: 0.5rem;
}

.detail-header h4 {
  margin: 0;
  font-size: 0.95rem;
  color: var(--accent-blue);
  font-weight: 600;
}

/* Loading & Empty States */
.loading-state, .empty-state {
  text-align: center;
  padding: 4rem;
  color: var(--text-muted);
  background: rgba(0,0,0,0.1);
  border-radius: 8px;
  border: 1px dashed var(--card-border);
}

@media (max-width: 1200px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 640px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }
}

/* Upgrade Card Style */
.upgrade-card {
  background: rgba(22, 27, 34, 0.85);
  border: 1px solid rgba(240, 140, 0, 0.4);
  box-shadow: 0 4px 20px rgba(240, 140, 0, 0.15);
}
.upgrade-card:hover {
  border-color: rgba(240, 140, 0, 0.8);
  box-shadow: 0 6px 25px rgba(240, 140, 0, 0.25);
}
.upgrade-glow {
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(240, 140, 0, 0.06) 0%, transparent 65%);
  pointer-events: none;
  animation: rotateGlow 20s linear infinite;
}
.upgrade-card h3 {
  color: #ff9800;
  text-shadow: 0 0 8px rgba(255, 152, 0, 0.2);
}
.upgrade-action-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
  margin-top: 6px;
}
.btn-upgrade-now {
  background: linear-gradient(135deg, #ff9800, #f57c00);
  border: none;
  color: white;
  padding: 5px 12px;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: bold;
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(245, 124, 0, 0.3);
  transition: all 0.2s;
}
.btn-upgrade-now:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 15px rgba(245, 124, 0, 0.5);
}
.arrow-up-glow {
  font-size: 2.2rem;
  font-weight: bold;
  color: #ff9800;
  text-shadow: 0 0 10px rgba(255, 152, 0, 0.5);
  animation: bounceUp 2s infinite;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
@keyframes bounceUp {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

/* 5 Columns Layout when Upgrade is visible */
.metrics-grid.has-upgrade {
  grid-template-columns: repeat(5, 1fr);
}
@media (max-width: 1400px) {
  .metrics-grid.has-upgrade {
    grid-template-columns: repeat(3, 1fr);
  }
}
@media (max-width: 1024px) {
  .metrics-grid.has-upgrade {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (max-width: 640px) {
  .metrics-grid.has-upgrade {
    grid-template-columns: 1fr;
  }
}

/* Upgrade Overlay Modal */
.upgrade-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(13, 17, 23, 0.9);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}
.upgrade-modal {
  background: rgba(22, 27, 34, 0.85);
  border: 1px solid rgba(88, 166, 255, 0.2);
  border-radius: 16px;
  width: 90%;
  max-width: 460px;
  padding: 3rem 2rem;
  text-align: center;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  animation: fadeIn 0.3s ease;
}
.upgrade-spinner {
  width: 50px;
  height: 50px;
  border: 4px solid rgba(88, 166, 255, 0.1);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  margin: 0 auto 1.5rem auto;
  animation: spin 1s linear infinite;
  box-shadow: 0 0 15px rgba(88, 166, 255, 0.2);
}
@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
.upgrade-modal h3 {
  margin: 0 0 1rem 0;
  font-size: 1.25rem;
  color: var(--text-main);
}
.upgrade-step {
  color: var(--accent-blue);
  font-weight: bold;
  font-size: 0.95rem;
  margin-bottom: 1rem;
}
.upgrade-tips {
  color: var(--text-muted);
  font-size: 0.8rem;
  line-height: 1.5;
}
</style>
