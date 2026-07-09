<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content chart-modal-content">
      <div class="modal-header">
        <h3>矿机算力曲线 - {{ worker }} ({{ ip }})</h3>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <HashrateChart :historyData="chartData" :title="'矿机 ' + worker + ' 实时算力'" />
        <div v-if="loading" class="loading-overlay">加载中...</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

import HashrateChart from './HashrateChart.vue'

const props = defineProps({
  ip: String,
  worker: String
})

const emit = defineEmits(['close'])

const chartData = ref([])
const loading = ref(true)
let intervalId = null

const fetchChartData = async () => {
  if (!props.ip) return
  try {
    const res = await fetch(`/api/miner/${props.ip}/history`)
    if (res.ok) {
      chartData.value = await res.json() || []
    }
  } catch (e) {
    console.error('Failed to load miner chart data', e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchChartData()
  intervalId = setInterval(fetchChartData, 60000) // update every minute
})

onUnmounted(() => {
  if (intervalId) clearInterval(intervalId)
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 8px;
  width: 90%;
  max-width: 900px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 24px rgba(0,0,0,0.5);
}

.modal-header {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #30363d;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  color: #c9d1d9;
  font-size: 1.1rem;
}

.close-btn {
  background: none;
  border: none;
  color: #8b949e;
  font-size: 1.5rem;
  cursor: pointer;
  line-height: 1;
}

.close-btn:hover {
  color: #c9d1d9;
}

.modal-body {
  padding: 1rem;
  position: relative;
  min-height: 400px;
}

.loading-overlay {
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(22, 27, 34, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #58a6ff;
  font-weight: bold;
}
</style>
