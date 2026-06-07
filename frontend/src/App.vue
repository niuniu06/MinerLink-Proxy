<template>
  <div class="app-layout">
    <div class="topbar">
      <h1><em>Go-Proxy</em> 代理引擎后台</h1>
      <div class="ctrls">
        <div class="status-badge">运行中</div>
        <button class="btn-settings" @click="openGlobalSettings">⚙️ 参数热修改</button>
        <button class="btn-restart" @click="globalRestart">🔄 全局热重启</button>
      </div>
    </div>
    
    <div class="main-layout">
      <nav class="sidebar">
        <div class="sg">系统控制</div>
        <a href="#" :class="{ active: currentView === 'dashboard' }" @click.prevent="currentView = 'dashboard'"><span class="dot dot-get"></span> 端口总览</a>
        <a href="#" @click.prevent="openGlobalSettings"><span class="dot dot-post"></span> 面板设置</a>
        <div class="sg">集群管理</div>
        <a href="#" :class="{ active: currentView === 'logs' }" @click.prevent="currentView = 'logs'"><span class="dot dot-del"></span> 系统日志 (实时)</a>
        <a href="#"><span class="dot dot-del"></span> 批量更新 (开发中)</a>
      </nav>

      <div class="main-content">
        <Dashboard v-if="currentView === 'dashboard'" ref="dashboardRef" @edit-config="openEditModal" />
        <SystemLogs v-if="currentView === 'logs'" />
        <button class="fab" @click="openAddModal">+</button>
      </div>
    </div>

    <ConfigModal 
      v-if="showModal" 
      :initial-data="editingConfig"
      @close="closeModal" 
      @saved="onConfigSaved" 
    />

    <GlobalSettingsModal
      v-if="showGlobalSettings"
      @close="closeGlobalSettings"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Dashboard from './components/Dashboard.vue'
import SystemLogs from './components/SystemLogs.vue'
import ConfigModal from './components/ConfigModal.vue'
import GlobalSettingsModal from './components/GlobalSettingsModal.vue'

const currentView = ref('dashboard')
const dashboardRef = ref(null)
const showModal = ref(false)
const showGlobalSettings = ref(false)
const editingConfig = ref(null)

const openAddModal = () => {
  editingConfig.value = null
  showModal.value = true
}

const openEditModal = (cfg) => {
  editingConfig.value = cfg
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
}

const onConfigSaved = () => {
  closeModal()
  if (dashboardRef.value) {
    dashboardRef.value.refresh()
  }
}

const openGlobalSettings = () => {
  showGlobalSettings.value = true
}

const closeGlobalSettings = () => {
  showGlobalSettings.value = false
}

const globalRestart = async () => {
  if (!confirm('确定要全量热重启引擎吗？矿机不会掉线。')) return
  try {
    const res = await fetch('/api/system/restart', { method: 'POST' })
    if (res.ok) {
      alert('热重启信号已发送')
    }
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  background: rgba(13, 17, 23, 0.92);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--card-border);
}

.topbar h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
}
.topbar h1 em {
  font-style: normal;
  color: var(--accent-blue);
}

.topbar .ctrls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.status-badge {
  background: rgba(63, 185, 80, 0.15);
  color: var(--accent-green);
  border: 1px solid rgba(63, 185, 80, 0.3);
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
}

.btn-settings, .btn-restart {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  color: var(--text-main);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 12px;
  cursor: pointer;
  font-weight: 600;
  transition: border 0.2s;
}

.btn-settings:hover, .btn-restart:hover {
  border-color: var(--accent-blue);
}

.btn-restart {
  color: var(--accent-red);
}
.btn-restart:hover {
  border-color: var(--accent-red);
}

.main-layout {
  display: flex;
  flex: 1;
}

.sidebar {
  position: sticky;
  top: 50px;
  width: 240px;
  min-width: 240px;
  height: calc(100vh - 50px);
  overflow-y: auto;
  padding: 16px 12px;
  border-right: 1px solid var(--card-border);
  background: var(--bg-color);
  font-size: 13px;
}

.sidebar .sg {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  color: var(--text-muted);
  margin: 16px 0 6px 8px;
  font-weight: 600;
}
.sidebar .sg:first-child { margin-top: 4px; }

.sidebar a {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  margin: 2px 0;
  color: var(--text-muted);
  border-radius: 6px;
  transition: all 0.15s;
  text-decoration: none;
}
.sidebar a:hover, .sidebar a.active {
  background: var(--card-bg);
  color: var(--text-main);
}

.sidebar .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.dot-get { background: var(--accent-green); }
.dot-post { background: var(--accent-blue); }
.dot-del { background: var(--text-muted); }

.main-content {
  flex: 1;
  padding: 24px 36px;
  max-width: 1200px;
  position: relative;
}

.fab {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  width: 50px;
  height: 50px;
  border-radius: 25px;
  background: var(--accent-blue);
  color: white;
  border: none;
  font-size: 1.5rem;
  font-weight: bold;
  cursor: pointer;
  box-shadow: 0 4px 15px rgba(88, 166, 255, 0.3);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 90;
}

.fab:hover {
  transform: scale(1.1) rotate(90deg);
}
</style>
