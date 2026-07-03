<template>
  <div class="logs-container">
    <div class="logs-header">
      <h2>系统运行日志</h2>
      <div class="logs-actions">
        <button class="btn-clear" @click="clearLogs" :disabled="loading">
          🗑️ 清空当前日志
        </button>
        <button class="btn-refresh" @click="fetchLogs" :disabled="loading">
          {{ loading ? '刷新中...' : '手动刷新' }}
        </button>
        <button class="btn-download" @click="downloadLogs">
          ⬇️ 完整日志下载
        </button>
      </div>
    </div>
    
    <div class="terminal-window" ref="terminalWindow">
      <pre v-if="logs">{{ logs }}</pre>
      <div v-else class="empty-state">暂无日志或正在加载...</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const logs = ref('')
const loading = ref(false)
const terminalWindow = ref(null)
let pollInterval = null

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/logs/tail')
    if (res.ok) {
      logs.value = await res.text()
      // Auto scroll to bottom
      setTimeout(() => {
        if (terminalWindow.value) {
          terminalWindow.value.scrollTop = terminalWindow.value.scrollHeight
        }
      }, 50)
    }
  } catch (err) {
    console.error("Failed to fetch logs:", err)
  } finally {
    loading.value = false
  }
}

const downloadLogs = () => {
  window.open('/api/logs/download', '_blank')
}

const clearLogs = async () => {
  if (!confirm('确定要清空当前的运行日志吗？(不可恢复)')) return
  
  loading.value = true
  try {
    const res = await fetch('/api/logs/clear', { method: 'DELETE' })
    if (res.ok) {
      logs.value = ''
      await fetchLogs()
    } else {
      alert('清空日志失败')
    }
  } catch (err) {
    console.error("Failed to clear logs:", err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLogs()
})

onUnmounted(() => {
  // No interval to clear
})
</script>

<style scoped>
.logs-container {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 120px);
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.logs-header h2 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--text-color);
}

.logs-actions {
  display: flex;
  gap: 1rem;
}

button {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}

.btn-refresh {
  background: var(--surface-light);
  color: var(--text-color);
}

.btn-refresh:hover {
  background: #333;
}

.btn-clear {
  background: transparent;
  color: #ff5e5e;
  border: 1px solid #ff5e5e;
}

.btn-clear:hover {
  background: rgba(255, 94, 94, 0.1);
}

.btn-download {
  background: var(--primary-color);
  color: #fff;
}

.btn-download:hover {
  opacity: 0.9;
}

.terminal-window {
  flex: 1;
  background: #0a0a0a;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  padding: 1rem;
  overflow-y: auto;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  color: #00ff00;
  box-shadow: inset 0 0 20px rgba(0,0,0,0.5);
}

.terminal-window pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.empty-state {
  color: var(--text-muted);
  text-align: center;
  margin-top: 2rem;
  font-family: var(--font-family);
}
</style>
