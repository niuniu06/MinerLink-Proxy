<template>
  <div class="modal-overlay">
    <div class="modal-content">
      <h2>⚙️ 全局面板设置</h2>
      
      <div class="form-group">
        <label>控制台访问端口</label>
        <input type="number" v-model.number="form.webPort" placeholder="例如: 10005" />
        <small class="help-text">
          这是网页访问的主端口。出厂默认端口是 8080。为了后台服务器安全，强烈建议您修改为您自己知道的隐蔽端口。<br>
          <span style="color: #ff5e5e; font-weight: bold;">⚠️ 警告：修改端口并保存后，代理引擎将立刻断开当前连接并重启！</span><br>
          重启后，您必须在云厂商的“安全组”或系统防火墙中放行新端口，然后手动使用 `http://您的IP:新端口` 来访问。
        </small>
      </div>

      <div class="form-group">
        <label>管理员账号 (登录面板用)</label>
        <input type="text" v-model="form.adminAccount" placeholder="默认: admin" />
      </div>
      
      <div class="form-group">
        <label>管理员密码(登录面板)</label>
        <input type="text" v-model="form.adminPassword" placeholder="默认: admin" />
        <small class="help-text">
          修改账号密码后，代理引擎也将重启以应用新密码。
        </small>
      </div>

      <div class="form-group">
        <label>配置备份与恢复</label>
        <div style="display: flex; gap: 10px; margin-top: 5px;">
          <button style="flex: 1; padding: 10px; border-radius: 6px; background-color: #2196F3; color: white; border: none; cursor: pointer; font-weight: bold;" @click="exportConfig">📥 一键备份 (导出所有端口及全局配置)</button>
          <button style="flex: 1; padding: 10px; border-radius: 6px; background-color: #ff9800; color: white; border: none; cursor: pointer; font-weight: bold;" @click="triggerRestore">📤 恢复配置 (导入备份文件)</button>
          <input type="file" ref="fileInput" accept=".json" style="display: none" @change="importConfig" />
        </div>
        <small class="help-text">
          恢复配置将会覆盖当前的所有端口，并且系统会自动重启以应用新的端口配置。
        </small>
      </div>

      <div class="modal-actions">
        <button class="btn-cancel" @click="$emit('close')" :disabled="saving">取消</button>
        <button class="btn-save" @click="saveConfig" :disabled="saving">
          {{ saving ? '保存并重启中...' : '保存修改 (触发重启)' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const emit = defineEmits(['close', 'saved'])

const form = ref({
  webPort: 0,
  enableLogging: true,
  adminAccount: 'admin',
  adminPassword: 'admin'
})

const saving = ref(false)
const fileInput = ref(null)

const triggerRestore = () => {
  if (fileInput.value) {
    fileInput.value.click()
  }
}

const exportConfig = async () => {
  try {
    const res = await fetch('/api/system/backup')
    if (res.ok) {
      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.style.display = 'none'
      a.href = url
      const now = new Date();
      const dateStr = now.toISOString().slice(0, 10).replace(/-/g, '');
      a.download = `MinerLink_Backup_${dateStr}.json`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
    } else {
      alert('备份失败！')
    }
  } catch (e) {
    alert('备份失败！' + e)
  }
}

const importConfig = async (event) => {
  const file = event.target.files[0]
  if (!file) return

  if (!confirm('⚠️ 警告：导入配置将清空当前所有端口并覆盖全局设置！\n导入完成后代理将被强制重启！\n\n确定要继续吗？')) {
    event.target.value = ''
    return
  }

  saving.value = true
  try {
    const text = await file.text()
    let data;
    try {
      data = JSON.parse(text)
    } catch (e) {
      alert('备份文件格式不正确，解析失败！')
      saving.value = false
      return
    }

    const res = await fetch('/api/system/restore', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
    
    if (res.ok) {
      alert('导入成功！系统正在后台重启以应用新配置。页面即将刷新。')
      setTimeout(() => {
        window.location.reload()
      }, 2000)
    } else {
      const err = await res.json()
      alert('导入失败: ' + err.error)
      saving.value = false
    }
  } catch (e) {
    alert('导入指令已发送！系统可能正在重启。页面即将刷新。')
    setTimeout(() => {
        window.location.reload()
    }, 2000)
  }
  event.target.value = ''
}

const fetchConfig = async () => {
  try {
    const res = await fetch('/api/global')
    if (res.ok) {
      const data = await res.json()
      // If DB has 0 (uninitialized), use window.location.port or fallback
      if (data.webPort > 0) {
        form.value.webPort = data.webPort
      } else {
        const port = parseInt(window.location.port) || 80
        form.value.webPort = port
      }
      form.value.enableLogging = data.enableLogging !== false // default true
      if (data.adminAccount) form.value.adminAccount = data.adminAccount;
      if (data.adminPassword) form.value.adminPassword = data.adminPassword;
    }
  } catch (e) {
    console.error(e)
  }
}

const saveConfig = async () => {
  if (form.value.webPort <= 0 || form.value.webPort > 65535) {
    alert('请输入有效的端口号 (1-65535)')
    return
  }

  const confirmMsg = `确定要将控制台端口修改为 ${form.value.webPort} 吗？\n\n警告：系统将立刻重启，您当前的页面会断开连接！\n您需要确保防火墙已经放行 ${form.value.webPort} 端口！`
  if (!confirm(confirmMsg)) return

  saving.value = true
  try {
    const res = await fetch('/api/config/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value)
    })
    
    if (res.ok) {
      const targetUrl = `http://${window.location.hostname}:${form.value.webPort}/ui/`
      alert(`保存成功！\n代理引擎正在后台重启，请在几秒后手动访问新地址:\n\n${targetUrl}`)
      window.location.href = targetUrl
    } else {
      const err = await res.json()
      alert('保存失败: ' + err.error)
    }
  } catch (e) {
    // Expected to fail after fetch completes because the server instantly quits!
    const targetUrl = `http://${window.location.hostname}:${form.value.webPort}/ui/`
    alert(`保存成功！\n代理引擎正在后台热重启以应用新端口，请稍候手动访问新地址:\n\n${targetUrl}`)
    setTimeout(() => {
        window.location.href = targetUrl
    }, 3000)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.7);
  backdrop-filter: blur(5px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: var(--card-bg);
  padding: 2rem;
  border-radius: 16px;
  border: 1px solid var(--card-border);
  width: 100%;
  max-width: 500px;
  box-shadow: 0 10px 40px rgba(0,0,0,0.5);
}
h2 {
  margin-top: 0;
  margin-bottom: 1.5rem;
  color: var(--text-color);
}
.form-group {
  margin-bottom: 1.5rem;
}
label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-color);
}
input {
  width: 100%;
  padding: 0.75rem;
  background: rgba(255,255,255,0.05);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  color: var(--text-color);
  font-family: inherit;
  font-size: 1rem;
}
input:focus {
  outline: none;
  border-color: var(--primary-color);
}
.help-text {
  display: block;
  margin-top: 0.5rem;
  color: var(--text-muted);
  font-size: 0.85rem;
  line-height: 1.4;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  margin-top: 2rem;
}
button {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}
button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.btn-cancel {
  background: transparent;
  color: var(--text-muted);
  border: 1px solid var(--card-border);
}
.btn-cancel:hover:not(:disabled) {
  background: rgba(255,255,255,0.1);
  color: var(--text-color);
}
.btn-save {
  background: var(--primary-color);
  color: white;
}
.btn-save:hover:not(:disabled) {
  filter: brightness(1.1);
}

/* Switch Styles borrowed from ConfigModal */
.switch-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
  padding: 1rem;
  background: rgba(255,255,255,0.03);
  border-radius: 8px;
  border: 1px solid var(--card-border);
  transition: all 0.2s;
  margin-bottom: 1rem;
}
.switch-row:hover {
  background: rgba(255,255,255,0.06);
}
.switch-row input[type="checkbox"] {
  width: 18px;
  height: 18px;
  margin-top: 2px;
  accent-color: var(--primary-color);
  cursor: pointer;
}
.switch-info {
  flex: 1;
}
.switch-title {
  color: var(--text-color);
  font-weight: 500;
  margin-bottom: 4px;
}
.switch-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.4;
}
</style>
