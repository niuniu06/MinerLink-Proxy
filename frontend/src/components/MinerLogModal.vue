<template>
  <div class="modal-backdrop">
    <div class="modal-content log-modal">
      <div class="modal-header">
        <div class="header-left">
          <h2>📊 矿机探针日志：{{ worker }}</h2>
        </div>
        <div class="header-right">
          <button class="export-btn" @click="exportLogs">⬇ 导出所有日志 (TXT)</button>
          <button class="close-btn" @click="$emit('close')">✕</button>
        </div>
      </div>

      <div class="log-tabs">
        <button 
          :class="['tab-btn', { active: activeTab === 'general' }]" 
          @click="activeTab = 'general'">
          有效份额日志 (最近5分钟)
        </button>
        <button 
          :class="['tab-btn', { active: activeTab === 'error' }]" 
          @click="activeTab = 'error'">
          异常/拒绝日志 (24小时)
        </button>
      </div>

      <div class="log-viewer" ref="logViewer">
        <div v-if="loading && logsToDisplay.length === 0" class="loading">加载中...</div>
        <div v-else-if="logsToDisplay.length === 0" class="empty">暂无相关日志</div>
        
        <div 
          v-for="(log, idx) in logsToDisplay" 
          :key="idx" 
          :class="['log-line', log.type]"
        >
          <span class="log-time">[{{ formatTime(log.timestamp) }}]</span>
          <span class="log-msg">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

const props = defineProps({
  port: Number,
  worker: String
})

defineEmits(['close'])

const activeTab = ref('general')
const loading = ref(false)
const generalLogs = ref([])
const errorLogs = ref([])
const logViewer = ref(null)

let intervalId = null

const logsToDisplay = computed(() => {
  return activeTab.value === 'general' ? generalLogs.value : errorLogs.value
})

const fetchLogs = async () => {
  try {
    const res = await fetch(`/api/minerlogs?port=${props.port}&worker=${encodeURIComponent(props.worker)}`)
    if (res.ok) {
      const data = await res.json()
      
      const prevLength = logsToDisplay.value.length
      
      generalLogs.value = data.general || []
      errorLogs.value = data.error || []
      
      // Auto-scroll to bottom if new logs arrived
      if (logsToDisplay.value.length > prevLength) {
        scrollToBottom()
      }
    }
  } catch (e) {
    console.error(e)
  }
}

const scrollToBottom = async () => {
  await nextTick()
  if (logViewer.value) {
    logViewer.value.scrollTop = logViewer.value.scrollHeight
  }
}

watch(activeTab, () => {
  scrollToBottom()
})

onMounted(async () => {
  loading.value = true
  await fetchLogs()
  loading.value = false
  scrollToBottom()
  
  intervalId = setInterval(fetchLogs, 2000) // refresh every 2s
})

onUnmounted(() => {
  clearInterval(intervalId)
})

const formatTime = (ts) => {
  const d = new Date(ts)
  const pad = (n) => n.toString().padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const exportLogs = () => {
  let content = `====================================================\n`;
  content += ` 矿机 (WORKER): ${props.worker}\n`;
  content += ` 导出时间: ${new Date().toLocaleString()}\n`;
  content += `====================================================\n\n`;

  content += `[异常/拒绝日志 - 包含底层抓包]\n`;
  content += `----------------------------------------------------\n`;
  if (errorLogs.value.length === 0) {
    content += `暂无异常日志\n`;
  } else {
    errorLogs.value.forEach(log => {
      content += `[${formatTime(log.timestamp)}] ${log.message}\n`;
    });
  }
  
  content += `\n\n[有效份额日志]\n`;
  content += `----------------------------------------------------\n`;
  if (generalLogs.value.length === 0) {
    content += `暂无有效份额日志\n`;
  } else {
    generalLogs.value.forEach(log => {
      content += `[${formatTime(log.timestamp)}] ${log.message}\n`;
    });
  }

  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `miner_logs_${props.worker}_${new Date().getTime()}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0,0,0,0.7);
  backdrop-filter: blur(4px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.log-modal {
  width: 900px;
  max-width: 95vw;
  height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-content {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0,0,0,0.5);
  overflow: hidden;
}

.modal-header {
  padding: 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--card-border);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: #fff;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.export-btn {
  background: var(--accent-cyan);
  color: #000;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-weight: bold;
  cursor: pointer;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  transition: all 0.2s;
}

.export-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,255,255,0.3);
}

.close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 1.5rem;
  cursor: pointer;
}

.close-btn:hover {
  color: #fff;
}

.log-tabs {
  display: flex;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  padding: 0 1.5rem;
  gap: 1rem;
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  padding: 1rem 0;
  cursor: pointer;
  font-size: 0.95rem;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #fff;
}

.tab-btn.active {
  color: var(--accent-cyan);
  border-bottom-color: var(--accent-cyan);
}

.log-viewer {
  flex: 1;
  background: #0d1117;
  padding: 1rem;
  overflow-y: auto;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 0.85rem;
}

.log-line {
  margin-bottom: 0.3rem;
  line-height: 1.4;
  word-break: break-all;
}

.log-time {
  color: #6e7681;
  margin-right: 0.8rem;
}

.log-msg {
  color: #c9d1d9;
}

.log-line.error .log-time,
.log-line.error .log-msg {
  color: var(--accent-red);
}

.loading, .empty {
  text-align: center;
  color: var(--text-muted);
  padding: 4rem;
}
</style>
