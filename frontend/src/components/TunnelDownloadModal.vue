<template>
  <div class="modal-overlay">
    <div class="modal-content">
      <div class="modal-header">
        <h2>📥 动态生成定制版防封客户端</h2>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>

      <div class="modal-body">
        <p class="desc">
          我们采用了高级二进制动态注入技术。您只需在下方填写矿场对应的服务器 IP 和各个币种的挖矿端口映射，系统将在一毫秒内为您打包出一个<strong>专属的单文件客户端</strong>。矿场客户下载后直接双击即可使用，真正零配置！
        </p>
        
        <div class="mappings-container">
          <label>端口映射配置表</label>
          <div v-for="(m, idx) in mappings" :key="idx" class="mapping-row">
            <input type="text" v-model="m.local" placeholder="本地矿机连的端口, 例如 :3333" class="small-input" />
            <span class="arrow-icon">👉</span>
            <input type="text" v-model="m.remote" placeholder="云端服务器地址, 例如 18.181.235.155:10101" />
            <button class="btn-remove" @click="removeMapping(idx)" v-if="mappings.length > 1">×</button>
          </div>
          <button class="btn-add" @click="addMapping">+ 添加更多端口映射</button>
        </div>

        <div class="script-block" v-if="mappings.length > 0 && mappings[0].remote">
          <label>Linux 矿场一键部署/更新脚本 (自动杀旧换新)：</label>
          <div class="code-wrap">
            <textarea readonly :value="wgetCommand"></textarea>
            <button class="btn-copy" @click="copyScript">复制</button>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn btn-primary" @click="downloadClient('windows')">🪄 生成 Windows 版 (.exe)</button>
        <button class="btn btn-linux" @click="downloadClient('linux')">🐧 生成 Linux 版</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const emit = defineEmits(['close'])

const mappings = ref([
  { local: ':3333', remote: '' }
])

onMounted(() => {
  const host = window.location.hostname
  if (host && host !== 'localhost' && host !== '127.0.0.1') {
    mappings.value[0].remote = `${host}:10101`
  }
})

const addMapping = () => {
  mappings.value.push({ local: '', remote: '' })
}

const removeMapping = (idx) => {
  if (mappings.value.length > 1) {
    mappings.value.splice(idx, 1)
  }
}

const wgetCommand = computed(() => {
  if (mappings.value.length === 0 || !mappings.value[0].remote) return '请先输入服务器地址...'
  const host = window.location.host
  const proto = window.location.protocol
  const mappingsJson = encodeURIComponent(JSON.stringify(mappings.value))
  const downloadUrl = `${proto}//${host}/api/download/custom?os=linux&mappings=${mappingsJson}`
  return `killall -9 go-xy 2>/dev/null; rm -f go-xy tunnel.log; wget -O go-xy "${downloadUrl}" && chmod +x go-xy && nohup ./go-xy > tunnel.log 2>&1 &`
})

const copyScript = async () => {
  try {
    await navigator.clipboard.writeText(wgetCommand.value)
    alert('一键部署脚本已复制到剪贴板！去 Linux 矿机上粘贴执行即可！')
  } catch (err) {
    alert('复制失败，请手动选中复制')
  }
}

const downloadClient = async (os) => {
  const validMappings = mappings.value.filter(m => m.remote && m.local)
  if (validMappings.length === 0) {
    alert('请至少填写一组完整的端口映射配置！')
    return
  }

  try {
    const res = await fetch('/api/download/custom', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        mappings: validMappings,
        os: os
      })
    })

    if (!res.ok) {
      const err = await res.json()
      alert('生成失败: ' + err.error)
      return
    }

    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = os === 'windows' ? 'go-xy.exe' : 'go-xy'
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.URL.revokeObjectURL(url)

    emit('close')
  } catch (err) {
    console.error(err)
    alert('网络错误，无法下载客户端')
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; width: 100vw; height: 100vh;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(5px);
}
.modal-content {
  background: var(--card-bg);
  width: 500px;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  overflow: hidden;
  animation: modalIn 0.2s ease-out forwards;
}
.modal-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.modal-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-color);
}
.close-btn {
  background: none; border: none;
  color: var(--text-muted);
  font-size: 1.5rem;
  cursor: pointer;
}
.close-btn:hover { color: var(--text-color); }

.modal-body {
  padding: 20px;
}
.desc {
  font-size: 0.9rem;
  color: var(--primary-color);
  background: rgba(30, 200, 160, 0.1);
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 20px;
  line-height: 1.4;
}

.mappings-container {
  margin-bottom: 15px;
  background: rgba(0, 0, 0, 0.1);
  padding: 15px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
}
.mappings-container label {
  display: block;
  margin-bottom: 12px;
  color: var(--text-color);
  font-weight: 600;
  font-size: 0.95rem;
}
.mapping-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.mapping-row input {
  flex: 1;
  padding: 8px 10px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-color);
  font-family: monospace;
}
.mapping-row input.small-input {
  flex: 0.6;
}
.mapping-row input:focus {
  outline: none;
  border-color: var(--primary-color);
}
.arrow-icon {
  font-size: 1rem;
}
.btn-remove {
  background: rgba(255, 60, 60, 0.2);
  color: #ff4d4d;
  border: 1px solid rgba(255, 60, 60, 0.3);
  border-radius: 4px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-remove:hover {
  background: rgba(255, 60, 60, 0.4);
}
.btn-add {
  margin-top: 10px;
  width: 100%;
  padding: 8px;
  background: transparent;
  border: 1px dashed var(--primary-color);
  color: var(--primary-color);
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}
.btn-add:hover {
  background: rgba(30, 200, 160, 0.1);
}

.script-block {
  margin-top: 20px;
  background: rgba(0, 0, 0, 0.2);
  padding: 15px;
  border-radius: 8px;
  border: 1px dashed var(--border-color);
}
.script-block label {
  display: block;
  font-size: 0.85rem;
  color: var(--accent-green);
  margin-bottom: 8px;
  font-weight: bold;
}
.code-wrap {
  position: relative;
}
.code-wrap textarea {
  width: 100%;
  height: 65px;
  background: #1e1e1e;
  color: #d4d4d4;
  border: 1px solid #333;
  border-radius: 6px;
  padding: 10px;
  font-family: monospace;
  font-size: 0.85rem;
  resize: none;
}
.btn-copy {
  position: absolute;
  right: 8px;
  bottom: 12px;
  background: var(--accent-blue);
  color: #fff;
  border: none;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 0.8rem;
  cursor: pointer;
  opacity: 0.8;
}
.btn-copy:hover {
  opacity: 1;
}

.modal-footer {
  padding: 15px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.btn {
  padding: 8px 16px;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s;
}
.btn-secondary {
  background: var(--bg-color);
  color: var(--text-muted);
  border: 1px solid var(--border-color);
}
.btn-secondary:hover { color: var(--text-color); background: rgba(255,255,255,0.05); }
.btn-primary {
  background: var(--primary-color);
  color: #fff;
}
.btn-primary:hover { filter: brightness(1.1); }
.btn-linux {
  background: #f39c12;
  color: #fff;
}
.btn-linux:hover { filter: brightness(1.1); }

@keyframes modalIn {
  from { opacity: 0; transform: translateY(20px) scale(0.95); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
</style>
