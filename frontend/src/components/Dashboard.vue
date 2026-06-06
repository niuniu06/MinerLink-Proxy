<template>
  <div class="dashboard">
    <div v-if="loading" class="loading">加载中...</div>
    <div v-if="!loading && configs.length === 0" class="empty">暂无配置，请点击右下角添加</div>
    
    <ProxyCard 
      v-for="cfg in configs" 
      :key="cfg.listenPort" 
      :config="cfg" 
      :stats="getStatsForPort(cfg.listenPort)"
      @edit="$emit('edit-config', cfg)"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import ProxyCard from './ProxyCard.vue'

const configs = ref([])
const stats = ref([])
const loading = ref(true)
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

const refresh = () => {
  fetchConfig()
  fetchStats()
}

defineExpose({ refresh })

const getStatsForPort = (port) => {
  return stats.value.find(s => s.listenPort === port) || null
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
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}
.loading, .empty {
  text-align: center;
  color: var(--text-muted);
  padding: 4rem;
  background: var(--card-bg);
  border-radius: 12px;
  border: 1px solid var(--card-border);
}
</style>
