<template>
  <div class="event-log-container glass-panel">
    <div class="log-header">
      <h3 class="log-title">
        <svg class="log-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
        关键事件日志
      </h3>
      <div class="log-badge">{{ events.length }} 条最新</div>
    </div>
    
    <div class="log-body" ref="logBodyRef">
      <div v-if="events.length === 0" class="empty-log">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
        <p>暂无连接异常事件</p>
      </div>
      
      <transition-group name="list" tag="ul" class="event-list">
        <li v-for="evt in events" :key="evt.id || evt.timestamp + evt.minerWorker" class="event-item">
          <div class="event-time">[{{ formatTime(evt.timestamp) }}]</div>
          <div class="event-content">
            <span class="event-worker">{{ evt.minerWorker }}</span>
            <span 
              class="event-badge" 
              :class="evt.eventType === 'ONLINE' ? 'badge-success' : 'badge-danger'"
            >
              {{ evt.eventType === 'ONLINE' ? '重新上线' : '断开连接' }}
            </span>
          </div>
        </li>
      </transition-group>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue'

const props = defineProps({
  events: {
    type: Array,
    default: () => []
  }
})

const logBodyRef = ref(null)

// Auto-scroll to top when new events arrive
watch(() => props.events, () => {
  nextTick(() => {
    if (logBodyRef.value) {
      logBodyRef.value.scrollTop = 0
    }
  })
}, { deep: true })

const formatTime = (ts) => {
  const d = new Date(ts)
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`
}
</script>

<style scoped>
.event-log-container {
  width: 100%;
  height: 385px; /* Matches chart height + header roughly */
  display: flex;
  flex-direction: column;
  padding: 1.25rem;
  border-radius: 16px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.05);
  margin-bottom: 2rem;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.log-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.1rem;
  font-weight: 600;
  color: #f8fafc;
  margin: 0;
}

.log-icon {
  width: 18px;
  height: 18px;
  color: #fbbf24;
}

.log-badge {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 12px;
  font-weight: 600;
}

.log-body {
  flex: 1;
  overflow-y: auto;
  padding-right: 5px;
}

/* Scrollbar styles */
.log-body::-webkit-scrollbar {
  width: 6px;
}
.log-body::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.5);
  border-radius: 3px;
}
.log-body::-webkit-scrollbar-thumb {
  background: rgba(99, 102, 241, 0.5);
  border-radius: 3px;
}
.log-body::-webkit-scrollbar-thumb:hover {
  background: rgba(99, 102, 241, 0.8);
}

.empty-log {
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: #64748b;
  gap: 1rem;
}
.empty-log svg {
  width: 48px;
  height: 48px;
  opacity: 0.5;
}

.event-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.event-item {
  display: flex;
  align-items: center;
  background: rgba(30, 41, 59, 0.5);
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  font-family: monospace;
  font-size: 0.85rem;
  border-left: 3px solid transparent;
  transition: all 0.3s;
}

.event-item:hover {
  background: rgba(30, 41, 59, 0.8);
}

.event-time {
  color: #94a3b8;
  margin-right: 0.75rem;
  min-width: 75px;
}

.event-content {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  overflow: hidden;
}

.event-worker {
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-badge {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: bold;
  font-family: sans-serif;
  margin-left: 0.5rem;
  white-space: nowrap;
}

.badge-success {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.badge-danger {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

/* List Transitions */
.list-enter-active,
.list-leave-active {
  transition: all 0.5s ease;
}
.list-enter-from {
  opacity: 0;
  transform: translateX(-30px);
}
.list-leave-to {
  opacity: 0;
  transform: translateX(30px);
}
</style>
