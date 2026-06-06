<template>
  <div class="app-container">
    <header class="header">
      <div class="title-wrap">
        <h1 class="title">透明抽水中转引擎</h1>
        <div class="status-badge"></div>
      </div>
      <div class="subtitle-wrap">
        <span>全协议并发支持·无限横向扩展</span>
        <button class="btn-settings" @click="openGlobalSettings">⚙️ 面板设置</button>
        <button class="btn-restart" @click="globalRestart">🔄 全局热重启</button>
      </div>
    </header>

    <Dashboard ref="dashboardRef" @edit-config="openEditModal" />

    <button class="fab" @click="openAddModal">+</button>

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
import ConfigModal from './components/ConfigModal.vue'
import GlobalSettingsModal from './components/GlobalSettingsModal.vue'

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
.btn-settings {
  background: transparent;
  color: var(--text-muted);
  border: 1px solid var(--card-border);
  padding: 0.5rem 1rem;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.9rem;
}
.btn-settings:hover {
  background: rgba(255,255,255,0.1);
  color: var(--text-color);
}
</style>
