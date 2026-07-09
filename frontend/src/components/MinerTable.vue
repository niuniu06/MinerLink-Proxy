<template>
  <div class="miner-table-wrap">
    <div class="table-header">
      <span class="total-info">共 {{ total }} 台矿机在线</span>
      <div class="pagination-controls">
        <select v-model="limit" @change="onLimitChange" class="page-select">
          <option :value="12">12条/页</option>
          <option :value="20">20条/页</option>
          <option :value="50">50条/页</option>
          <option :value="100">100条/页</option>
        </select>
        <button :disabled="page <= 1" @click="page--; fetchMiners()" class="page-btn">上一页</button>
        <span class="page-info">{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="page++; fetchMiners()" class="page-btn">下一页</button>
      </div>
    </div>
    
    <table class="miner-table">
      <thead>
        <tr>
          <th>矿机 (WORKER)</th>
          <th>算力 (HASHRATE)</th>
          <th>提交 (SUBMITS)</th>
          <th>抽水拦截 (FEE)</th>
          <th>当前难度 (DIFF)</th>
          <th>在线时间 (UPTIME)</th>
          <th>钱包 (WALLET)</th>
          <th>操作 (ACTIONS)</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading && miners.length === 0">
          <td colspan="8" class="empty-row">加载中...</td>
        </tr>
        <tr v-else-if="miners.length === 0">
          <td colspan="8" class="empty-row">暂无在线矿机</td>
        </tr>
        <tr v-for="miner in miners" :key="miner.id" :class="{ 'offline-row': miner.isOffline }">
          <td class="worker-cell">
            <div class="worker-name-row">
              <span v-if="miner.isEncrypted" class="secure-icon" title="隧道加密">🛡️</span>
              <span class="worker-name">{{ miner.worker || 'worker' }}</span>
              <span v-if="miner.isOffline" class="offline-badge">离线</span>
            </div>
            <div v-if="miner.clientAgent" class="client-agent">
              {{ miner.clientAgent }}
            </div>
          </td>
          <td class="hashrate">{{ miner.hashrate }}</td>
          <td class="submits-cell">
            <div class="valid">有效 {{ formatSubmits(miner.validShares) }}</div>
            <div class="invalid">无效 {{ formatSubmits(miner.invalidShares) }}</div>
          </td>
          <td class="fee">{{ miner.feeShares }}</td>
          <td class="diff">{{ miner.currentDiff ? miner.currentDiff.toFixed(2) : '...' }}</td>
          <td>{{ formatUptime(miner.uptime) }}</td>
          <td class="wallet">{{ maskWallet(miner.wallet) }}</td>
          <td class="actions-cell">
            <button class="log-btn" @click="showLogs(miner.worker)">日志</button>
            <button class="log-btn chart-btn" @click="showChart(miner)">曲线</button>
          </td>
        </tr>
      </tbody>
    </table>
    
    <MinerLogModal 
      v-if="activeLogWorker" 
      :port="port" 
      :worker="activeLogWorker" 
      @close="activeLogWorker = null" 
    />

    <MinerChartModal
      v-if="activeChartMiner"
      :ip="activeChartMiner.id"
      :worker="activeChartMiner.worker"
      @close="activeChartMiner = null"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import MinerLogModal from './MinerLogModal.vue'
import MinerChartModal from './MinerChartModal.vue'

const props = defineProps({
  port: Number
})

const miners = ref([])
const total = ref(0)
const page = ref(1)
const limit = ref(12)
const loading = ref(false)
const activeLogWorker = ref(null)
const activeChartMiner = ref(null)

let intervalId = null

const totalPages = computed(() => {
  return Math.max(1, Math.ceil(total.value / limit.value))
})

const fetchMiners = async () => {
  try {
    const res = await fetch(`/api/miners?port=${props.port}&page=${page.value}&limit=${limit.value}`)
    if (res.ok) {
      const data = await res.json()
      miners.value = data.miners || []
      total.value = data.total || 0
      
      // Auto adjust page if out of bounds
      if (page.value > totalPages.value && totalPages.value > 0) {
        page.value = totalPages.value
        await fetchMiners()
      }
    }
  } catch (e) {
    console.error(e)
  }
}

const onLimitChange = () => {
  page.value = 1
  fetchMiners()
}

const showLogs = (worker) => {
  activeLogWorker.value = worker || 'default'
}

const showChart = (miner) => {
  activeChartMiner.value = miner
}

onMounted(() => {
  loading.value = true
  fetchMiners().then(() => loading.value = false)
  intervalId = setInterval(fetchMiners, 30000)
})

onUnmounted(() => {
  clearInterval(intervalId)
})

watch(() => props.port, () => {
  page.value = 1
  fetchMiners()
})

const formatUptime = (secs) => {
  if (!secs) return '0s'
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  let res = ''
  if (h > 0) res += `${h}h `
  if (m > 0 || h > 0) res += `${m}m `
  res += `${s}s`
  return res
}

const maskWallet = (wallet) => {
  if (!wallet || wallet.length < 10) return wallet
  return wallet.substring(0, 5) + '...' + wallet.substring(wallet.length - 4)
}

const formatSubmits = (val) => {
  if (val == null) return "0"
  if (val >= 1000) {
    return (val / 1000).toFixed(2) + 'K'
  }
  return val.toString()
}
</script>

<style scoped>
.miner-table-wrap {
  overflow-x: auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 0.5rem;
}

.total-info {
  color: var(--text-muted);
  font-size: 0.9rem;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 0.8rem;
}

.page-select {
  background: var(--bg-dark);
  color: #fff;
  border: 1px solid var(--card-border);
  padding: 0.3rem 0.5rem;
  border-radius: 4px;
  outline: none;
}

.page-btn {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--card-border);
  color: #fff;
  padding: 0.3rem 0.8rem;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.1);
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 0.9rem;
  color: var(--text-muted);
}

.miner-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.miner-table th {
  color: var(--text-muted);
  font-size: 0.8rem;
  padding: 1rem 0.5rem;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  font-weight: normal;
}

.miner-table td {
  padding: 1rem 0.5rem;
  border-bottom: 1px solid rgba(255,255,255,0.02);
  font-size: 0.9rem;
}

.worker-cell {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.worker-name-row {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.worker-name {
  color: #f3f4f6;
  font-weight: bold;
}

.client-agent {
  color: #9ca3af;
  font-size: 0.75rem;
  margin-top: 0.2rem;
  font-family: monospace;
}

.secure-icon {
  font-size: 1.1rem;
}

.hashrate {
  color: var(--accent-cyan);
  font-weight: bold;
}

.submits-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 0.85rem;
}

.valid {
  color: var(--accent-green);
}

.invalid {
  color: var(--accent-red);
}

.fee {
  color: var(--accent-blue);
}

.wallet {
  background: rgba(255,255,255,0.05);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-family: monospace;
  color: #bbb;
}

.log-btn {
  background: var(--accent-blue);
  color: #fff;
  border: none;
  padding: 0.3rem 0.6rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
  transition: opacity 0.2s;
}

.log-btn:hover {
  opacity: 0.8;
}

.empty-row {
  text-align: center;
  padding: 2rem !important;
  color: var(--text-muted);
}

.offline-row {
  opacity: 0.5;
  filter: grayscale(100%);
}

.offline-badge {
  background-color: #ff4d4f;
  color: white;
  font-size: 0.75rem;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 8px;
  font-weight: bold;
}
</style>
