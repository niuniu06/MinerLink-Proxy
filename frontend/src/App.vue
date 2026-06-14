<template>
  <div v-if="!isLoggedIn">
    <Login @success="onLoginSuccess" />
  </div>
  <div class="app-layout" v-else>
    <div class="topbar">
      <h1><em>MinerLink-Proxy</em> 控制台 <span class="version-label" v-if="sysStatus && sysStatus.version">{{ sysStatus.version }}</span></h1>
      <div class="ctrls">
        <div class="sys-metrics" v-if="sysStatus">
          <div class="metric-item" :class="{ warning: sysStatus.cpuPercent > 80 }">
            <span class="m-label">CPU</span>
            <span class="m-val">{{ sysStatus.cpuPercent }}%</span>
          </div>
          <div class="metric-item has-tooltip" :class="{ warning: sysStatus.memoryPercent > 85 }" @mouseenter="showMemTooltip = true" @mouseleave="showMemTooltip = false" @click="showMemTooltip = !showMemTooltip">
            <span class="m-label">内存</span>
            <span class="m-val">{{ sysStatus.memoryPercent }}%</span>
            <div class="mem-tooltip" v-show="showMemTooltip">
              <div class="tt-header">总内存详细信息</div>
              <div class="tt-row">
                <span class="tt-label">物理内存:</span>
                <span class="tt-val">{{ sysStatus.totalMemMB ? Math.round(sysStatus.totalMemMB) : 0 }}Mb</span>
              </div>
              <div class="tt-row">
                <span class="tt-label">系统内存:</span>
                <span class="tt-val">{{ sysStatus.totalMemMB ? Math.round(sysStatus.totalMemMB * (sysStatus.memoryPercent / 100)) : 0 }}Mb ({{ sysStatus.memoryPercent }}%)</span>
              </div>
              <div class="tt-row">
                <span class="tt-label">程序占用内存:</span>
                <span class="tt-val">{{ sysStatus.procMemMB }}Mb ({{ sysStatus.totalMemMB ? ((sysStatus.procMemMB / sysStatus.totalMemMB) * 100).toFixed(2) : 0 }}%)</span>
              </div>
            </div>
          </div>
          <div class="metric-item uptime">
            <span class="m-label">已运行</span>
            <span class="m-val">{{ formatUptime(sysStatus.uptimeSeconds) }}</span>
          </div>
        </div>
        <div class="status-badge">运行中</div>
        <button class="btn-settings" @click="openGlobalSettings">⚙️ 参数热修改</button>
        <button class="btn-restart" @click="globalLogout">🚪 退出登录</button>
        <button class="btn-restart" @click="globalRestart">🔄 全局热重启</button>
      </div>
    </div>
    
    <div class="main-layout">
      <nav class="sidebar">
        <div class="sg">系统控制</div>
        <a href="#" :class="{ active: currentView === 'dashboard' }" @click.prevent="currentView = 'dashboard'"><span class="dot dot-get"></span> 端口总览</a>
        <a href="#" @click.prevent="openGlobalSettings"><span class="dot dot-post"></span> 面板设置</a>
        <div class="sg">集群管理</div>
        <a href="#" @click.prevent="showTunnelModal = true"><span class="dot dot-get"></span> 隧道客户端 (一键定制)</a>
        <a href="#" :class="{ active: currentView === 'logs' }" @click.prevent="currentView = 'logs'"><span class="dot dot-del"></span> 系统日志 (实时)</a>
        <a href="#"><span class="dot dot-del"></span> 批量更新 (开发中)</a>
      </nav>

      <div class="main-content">
        <Dashboard v-if="currentView === 'dashboard'" ref="dashboardRef" :sys-status="sysStatus" :update-info="updateInfo" @edit-config="openEditModal" />
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

    <TunnelDownloadModal
      v-if="showTunnelModal"
      @close="showTunnelModal = false"
    />

    <GlobalSettingsModal
      v-if="showGlobalSettings"
      @close="closeGlobalSettings"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import Dashboard from './components/Dashboard.vue'
import SystemLogs from './components/SystemLogs.vue'
import ConfigModal from './components/ConfigModal.vue'
import GlobalSettingsModal from './components/GlobalSettingsModal.vue'
import Login from './components/Login.vue'

const showMemTooltip = ref(false)

const isLoggedIn = ref(false)

const onLoginSuccess = () => {
  isLoggedIn.value = true
  fetchSysStatus()
  checkUpdate()
  if (sysIntervalId) clearInterval(sysIntervalId)
  sysIntervalId = setInterval(fetchSysStatus, 2000)
}

const globalLogout = () => {
  localStorage.removeItem('mlp_token')
  sessionStorage.removeItem('mlp_token')
  isLoggedIn.value = false
  if (sysIntervalId) clearInterval(sysIntervalId)
}
import TunnelDownloadModal from './components/TunnelDownloadModal.vue'

const currentView = ref('dashboard')
const dashboardRef = ref(null)
const showModal = ref(false)
const showGlobalSettings = ref(false)
const showTunnelModal = ref(false)
const editingConfig = ref(null)

const sysStatus = ref({ cpuPercent: 0.0, memoryPercent: 0.0, uptimeSeconds: 0 })
const updateInfo = ref({ hasUpdate: false, currentVersion: '', latestVersion: '', changelog: '' })
let sysIntervalId = null

const fetchSysStatus = async () => {
  try {
    const res = await fetch('/api/system/status')
    if (res.ok) {
      sysStatus.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to fetch system status:', e)
  }
}

const checkUpdate = async () => {
  try {
    const urlParams = new URLSearchParams(window.location.search);
    const mockParam = urlParams.get('mock') ? '?mock=1' : '';
    const res = await fetch('/api/system/check_update' + mockParam)
    if (res.ok) {
      updateInfo.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to check for updates:', e)
  }
}

onMounted(() => {
  const token = localStorage.getItem('mlp_token') || sessionStorage.getItem('mlp_token')
  if (token) {
    isLoggedIn.value = true
    fetchSysStatus()
    checkUpdate()
    sysIntervalId = setInterval(fetchSysStatus, 2000)
  }
})

onUnmounted(() => {
  if (sysIntervalId) clearInterval(sysIntervalId)
})

const formatUptime = (secs) => {
  if (!secs) return '0分'
  const d = Math.floor(secs / 86400)
  const h = Math.floor((secs % 86400) / 3600)
  const m = Math.floor((secs % 3600) / 60)
  let res = ''
  if (d > 0) res += `${d}天 `
  if (h > 0 || d > 0) res += `${h}小时 `
  res += `${m}分`
  return res
}

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

.sys-metrics {
  display: flex;
  gap: 16px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 30px;
  padding: 4px 16px;
  margin-right: 8px;
  align-items: center;
  font-size: 12px;
}
.metric-item {
  display: flex;
  align-items: center;
  gap: 6px;
  position: relative;
}
.metric-item .m-label {
  color: var(--text-muted);
  font-size: 11px;
}
.metric-item .m-val {
  color: #fff;
  font-family: 'SF Mono', Consolas, monospace;
  font-weight: 600;
}
.has-tooltip {
  cursor: pointer;
}
.mem-tooltip {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-top: 10px;
  background: #1e2227;
  border: 1px solid rgba(88, 166, 255, 0.2);
  border-radius: 6px;
  padding: 12px;
  width: 260px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  z-index: 1000;
  cursor: default;
}
.tt-header {
  color: #ffb86c;
  font-weight: bold;
  font-size: 13px;
  margin-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding-bottom: 6px;
}
.tt-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
  font-size: 12px;
}
.tt-row:last-child {
  margin-bottom: 0;
}
.tt-label {
  color: #aab2c0;
}
.tt-val {
  color: #e5e9f0;
  font-family: 'SF Mono', Consolas, monospace;
}
.warning .m-val {
  color: var(--accent-red);
}
.metric-item.uptime .m-val {
  color: var(--accent-green);
}
.version-label {
  font-size: 10px;
  font-weight: 600;
  color: var(--accent-blue);
  background: rgba(88, 166, 255, 0.1);
  border: 1px solid rgba(88, 166, 255, 0.2);
  padding: 1px 6px;
  border-radius: 4px;
  margin-left: 8px;
  vertical-align: middle;
  font-family: 'SF Mono', Consolas, monospace;
}
</style>
