<template>
  <div class="miner-table-wrap">
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
        </tr>
      </thead>
      <tbody>
        <tr v-if="!miners || miners.length === 0">
          <td colspan="7" class="empty-row">暂无在线矿机</td>
        </tr>
        <tr v-for="miner in sortedMiners" :key="miner.id">
          <td class="worker-name">
            <span v-if="miner.isEncrypted" class="secure-icon" title="隧道加密">🛡️</span>
            {{ miner.worker || 'worker' }}
          </td>
          <td class="hashrate">{{ miner.hashrate }}</td>
          <td>
            <span class="valid">{{ miner.validShares }} 有效</span> | 
            <span class="invalid">{{ miner.invalidShares }} 无效</span>
          </td>
          <td class="fee">{{ miner.feeShares }}</td>
          <td class="diff">{{ miner.currentDiff ? miner.currentDiff.toFixed(2) : '...' }}</td>
          <td>{{ formatUptime(miner.uptime) }}</td>
          <td class="wallet">{{ maskWallet(miner.wallet) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  miners: Array
})

const sortedMiners = computed(() => {
  if (!props.miners) return []
  return [...props.miners].sort((a, b) => b.uptime - a.uptime)
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
</script>

<style scoped>
.miner-table-wrap {
  overflow-x: auto;
}

.miner-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.miner-table th {
  color: var(--text-muted);
  font-size: 0.8rem;
  padding: 1rem 0;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  font-weight: normal;
}

.miner-table td {
  padding: 1rem 0;
  border-bottom: 1px solid rgba(255,255,255,0.02);
  font-size: 0.9rem;
}

.worker-name {
  color: #f3f4f6;
  font-weight: bold;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.secure-icon {
  font-size: 1.1rem;
}

.hashrate {
  color: var(--accent-cyan);
  font-weight: bold;
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

.empty-row {
  text-align: center;
  padding: 2rem !important;
  color: var(--text-muted);
}
</style>
