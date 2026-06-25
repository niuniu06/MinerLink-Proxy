<template>
  <div class="modal-overlay">
    <div class="modal-content">
      <div class="modal-header">
        <h2>{{ isEdit ? '⚙ 参数热修改' : '➕ 添加新端口配置' }}</h2>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>

      <form @submit.prevent="save">
        <div class="form-grid">
          <div class="form-group">
            <label>监听端口 (PORT)</label>
            <input v-model="form.listenPort" type="number" required />
          </div>
          <div class="form-group">
            <label>币种名称 (COIN)</label>
            <select v-model="form.coinName" required class="coin-select">
              <option value="" disabled selected>请选择币种</option>
              <option value="BTC">BTC (比特币)</option>
              <option value="BCH">BCH</option>
              <option value="KAS">KAS (Kaspa)</option>
              <option value="LTC">LTC (莱特币)</option>
              <option value="DOGE">DOGE (狗狗币)</option>
              <option value="ETC">ETC (以太经典)</option>
              <option value="ETHW">ETHW</option>
              <option value="DASH">DASH</option>
              <option value="CKB">CKB</option>
              <option value="PRL">PRL (珍珠币 Perle)</option>
            </select>
          </div>
          <div class="form-group full-width">
            <label>主矿池地址 (MAIN POOL)</label>
            <div style="display: flex; gap: 8px; align-items: center;">
              <input v-model="form.poolAddress" required style="flex: 1;" />
              <button type="button" class="btn-test" @click="testPing('main')">✅ 测试连接</button>
            </div>
            <div v-if="pingResult.main" :class="['ping-result', pingResult.main.success ? 'success' : 'error']">
              {{ pingResult.main.text }}
            </div>
          </div>
          <div class="form-group full-width">
            <label>独立抽水矿池地址 (FEE POOL - 强烈建议留空，默认同主矿池)</label>
            <div style="display: flex; gap: 8px; align-items: center;">
              <input v-model="form.feePoolAddress" placeholder="留空则自动连接同主矿池服务器，网络最稳定" style="flex: 1;" />
              <button type="button" class="btn-test" @click="testPing('fee')" :disabled="!form.feePoolAddress">✅ 测试连接</button>
            </div>
            <div v-if="pingResult.fee" :class="['ping-result', pingResult.fee.success ? 'success' : 'error']">
              {{ pingResult.fee.text }}
            </div>
          </div>
          <div class="form-group full-width">
            <label>运营者抽水钱包 (OP WALLET - 留空不启用)</label>
            <input v-model="form.operatorWallet" placeholder="0x..." />
          </div>
          <div class="form-group">
            <label>运营者抽水比例 (%)</label>
            <input v-model="form.operatorFeePercent" type="number" step="0.1" />
          </div>
          <div class="form-group full-width">
            <label>抽水大周期总时长 (分钟) [留空则默认 100]</label>
            <input v-model="form.feeCycleMinutes" type="number" placeholder="100" />
            <div style="font-size: 12px; color: var(--text-muted); margin-top: 4px;">如设为 1440 且抽水 1%，则每 24 小时连续抽水 14.4 分钟。设为 10 且抽水 1%，则每 10 分钟抽 6 秒。</div>
          </div>
        </div>

        <div class="advanced-section">
          <button type="button" class="advanced-toggle" @click="showAdvanced = !showAdvanced">
            {{ showAdvanced ? '▼ 收起高级黑科技设定' : '▶ 展开高级黑科技设定' }}
          </button>
          
          <div v-if="showAdvanced" class="advanced-content">

            <label class="switch-row">
              <input type="checkbox" v-model="form.enableEthTargetRewrite" />
              <div class="switch-info">
                <div class="switch-title">💡 ETH/ETC 跨矿池难度无缝伪造技术</div>
                <div class="switch-desc">拦截并篡改抽水矿池下发的 Target Hash 使其与主矿池完全一致，彻底消除跨矿池算力断层</div>
              </div>
            </label>

            <label class="switch-row" v-show="true">
              <input type="checkbox" v-model="form.enableTcpNoDelay" />
              <div class="switch-info">
                <div class="switch-title">⚡ 强制底层极速网络推送 (TCP NoDelay)</div>
                <div class="switch-desc">物理级别压榨系统网卡缓存，极速将新区块任务推送至矿机控制板，减少换块延迟</div>
              </div>
            </label>


            <label class="switch-row">
              <input type="checkbox" v-model="form.enableAsic" />
              <div class="switch-info">
                <div class="switch-title">专业ASIC芯片机增强支持</div>
                <div class="switch-desc">强行修正 Extranonce 与难度下发，对抗各种固件拒绝率</div>
              </div>
            </label>

            <label class="switch-row" v-show="true">
              <input type="checkbox" v-model="form.enableAntiBan" />
              <div class="switch-info">
                <div class="switch-title">开启完美防封禁 (0拒绝)</div>
                <div class="switch-desc">强行拦截矿池所有的 Reject 报错，向矿机伪造 Accept 成功响应</div>
              </div>
            </label>

            <label class="switch-row" v-show="true">
              <input type="checkbox" v-model="form.enableDetailedLog" />
              <div class="switch-info">
                <div class="switch-title">开启底层原始抓包日志</div>
                <div class="switch-desc">在矿机日志中输出矿机原始提交报文与矿池拒绝报文，用于硬核排错排查</div>
              </div>
            </label>


            <label class="switch-row">
              <input type="checkbox" v-model="form.enableVardiff" />
              <div class="switch-info">
                <div class="switch-title">开启 Auto-Vardiff 智能动态难度 (通用版)</div>
                <div class="switch-desc">代理端自动接管难度，智能降频。无脑拦截低端矿机风暴，省 CPU 省带宽</div>
              </div>
            </label>

            <div class="form-group" style="margin-top: 1rem;" v-if="form.enableVardiff">
              <label>目标提交频率 (Shares/Min)</label>
              <input v-model="form.targetShareRate" type="number" placeholder="默认 2" />
              <div class="field-hint" style="font-size: 12px; color: var(--text-muted); margin-top: 4px;">代理会自动调整矿机本地难度，使每台矿机强制保持在每分钟提交 2 个 Share，绝对平均！</div>
            </div>

            <div class="form-group" style="margin-top: 1rem;">
              <label>主矿池强制初始难度 (Main Difficulty)</label>
              <input v-model="form.mainFixedDifficulty" placeholder="如 d=2048 或 auto，留空则不干预" />
            </div>

            <div class="form-group" style="margin-top: 1rem;">
              <label>抽水矿池强制初始难度 (Fee Difficulty)</label>
              <input v-model="form.feeFixedDifficulty" placeholder="强烈推荐填 auto 或 d=2048 保底" />
              <div class="field-hint" style="font-size: 12px; color: var(--text-muted); margin-top: 4px;">填 auto 自动继承主池当前难度，老机器填具体数字防掉线</div>
            </div>

            <div class="form-group" style="margin-top: 1rem;">
              <label>算力全局虚标倍率 (默认 1.0)</label>
              <input v-model="form.hashrateMultiplier" type="number" step="0.1" />
            </div>

            <div class="form-group" style="margin-top: 1rem;">
              <label>算力显示单位 (默认自动计算，例如 TH/s, GH/s, MH/s)</label>
              <input v-model="form.hashrateUnit" placeholder="留空则自动根据算力大小适配" />
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" class="btn-cancel" @click="$emit('close')">取消</button>
          <button type="button" v-if="isEdit" class="btn-delete" @click="deleteConfig">删除此端口</button>
          <button type="submit" class="btn-save">{{ isEdit ? '保存并热重载' : '部署启动' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  initialData: Object
})
const emit = defineEmits(['close', 'saved'])

const pingResult = ref({ main: null, fee: null })

const testPing = async (type) => {
  const addr = type === 'main' ? form.value.poolAddress : form.value.feePoolAddress;
  if (!addr) return;
  
  pingResult.value[type] = { success: true, text: '测速中...' };
  
  try {
    const res = await fetch('/api/system/ping', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ poolAddress: addr })
    });
    const data = await res.json();
    if (data.success) {
      pingResult.value[type] = { success: true, text: `✅ 连接成功, ${data.latencyMs.toFixed(1)}ms` };
    } else {
      pingResult.value[type] = { success: false, text: `❌ 连接失败 (${data.error})` };
    }
  } catch (e) {
    pingResult.value[type] = { success: false, text: `❌ 测试异常` };
  }
}

const isEdit = ref(false)
const showAdvanced = ref(false)

const form = ref({
  listenPort: '',
  coinName: '',
  poolAddress: '',
  devWallet: 'linkpro168',
  devFeePercent: 2.0,
  operatorWallet: '',
  operatorFeePercent: '',
  enableAsic: false,
  enableAntiBan: true,
  enableDetailedLog: false,
  mainFixedDifficulty: '',
  feeFixedDifficulty: '',
  hashrateMultiplier: 1.0,
  hashrateUnit: '',
  feeCycleMinutes: '',
  feePoolAddress: '',
  enableVardiff: false,
  targetShareRate: '',
  enableEthTargetRewrite: false,
  enableTcpNoDelay: true
})

onMounted(() => {
  if (props.initialData) {
    isEdit.value = true
    // Fill data
    form.value = {
      listenPort: props.initialData.listenPort,
      coinName: props.initialData.coinName || '',
      poolAddress: props.initialData.poolAddress || '',
      devWallet: props.initialData.devWallet || '',
      devFeePercent: props.initialData.devFeePercent || 0,
      operatorWallet: props.initialData.operatorWallet || '',
      operatorFeePercent: props.initialData.operatorFeePercent || 0,
      enableAsic: !!props.initialData.enableAsic,
      enableAntiBan: !!props.initialData.enableAntiBan,
      enableDetailedLog: !!props.initialData.enableDetailedLog,
      mainFixedDifficulty: props.initialData.mainFixedDifficulty || '',
      feeFixedDifficulty: props.initialData.feeFixedDifficulty || '',
      hashrateMultiplier: props.initialData.hashrateMultiplier || 1.0,
      hashrateUnit: props.initialData.hashrateUnit || '',
      feeCycleMinutes: props.initialData.feeCycleMinutes || '',
      feePoolAddress: props.initialData.feePoolAddress || '',
      enableVardiff: !!props.initialData.enableVardiff,
      targetShareRate: props.initialData.targetShareRate || '',
      enableEthTargetRewrite: !!props.initialData.enableEthTargetRewrite,
      enableTcpNoDelay: !!props.initialData.enableTcpNoDelay
    }
  }
})

const save = async () => {
  try {
    const res = await fetch('/api/config/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        isEdit: isEdit.value,
        oldListenPort: isEdit.value ? props.initialData.listenPort : 0,
        listenPort: Number(form.value.listenPort),
        coinName: form.value.coinName,
        poolAddress: form.value.poolAddress,
        feePoolAddress: form.value.feePoolAddress,
        devWallet: form.value.devWallet,
        devFeePercent: Number(form.value.devFeePercent),
        operatorWallet: form.value.operatorWallet,
        operatorFeePercent: Number(form.value.operatorFeePercent || 0),
        enableAsic: form.value.enableAsic,
        enableAntiBan: form.value.enableAntiBan,
        enableDetailedLog: form.value.enableDetailedLog,
        mainFixedDifficulty: form.value.mainFixedDifficulty,
        feeFixedDifficulty: form.value.feeFixedDifficulty,
        hashrateMultiplier: Number(form.value.hashrateMultiplier),
        hashrateUnit: form.value.hashrateUnit,
        feeCycleMinutes: Number(form.value.feeCycleMinutes || 100),
        enableVardiff: form.value.enableVardiff,
        targetShareRate: Number(form.value.targetShareRate || 2),
        enableEthTargetRewrite: form.value.enableEthTargetRewrite,
        enableTcpNoDelay: form.value.enableTcpNoDelay
      })
    })
    if (res.ok) {
      emit('saved')
    } else {
      const data = await res.json()
      alert('保存失败: ' + data.error)
    }
  } catch (e) {
    alert('网络错误')
  }
}

const deleteConfig = async () => {
  if (!confirm('确定要彻底删除该端口配置及所有矿机吗？')) return
  try {
    const res = await fetch('/api/config/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ listenPort: Number(form.value.listenPort) })
    })
    if (res.ok) {
      emit('saved')
    } else {
      alert('删除失败')
    }
  } catch(e) {
    alert('网络错误')
  }
}
</script>

<style scoped>
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}
.modal-header h2 {
  margin: 0;
  color: var(--accent-blue);
  font-size: 1.4rem;
}
.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.5rem;
  cursor: pointer;
}
.close-btn:hover { color: white; }

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}
.full-width {
  grid-column: 1 / -1;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.form-group label {
  font-size: 0.85rem;
  color: var(--text-muted);
}
input, select {
  background: rgba(0,0,0,0.3);
  border: 1px solid rgba(255,255,255,0.1);
  color: white;
  padding: 0.6rem;
  border-radius: 6px;
}
input:focus, select:focus {
  outline: none;
  border-color: var(--accent-cyan);
}
select option {
  background: #2a2a2a;
  color: white;
}
input:disabled, select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.advanced-section {
  margin-top: 1.5rem;
  border-top: 1px solid rgba(255,255,255,0.05);
  padding-top: 1rem;
}
.advanced-toggle {
  background: transparent;
  color: var(--accent-green);
  border: none;
  padding: 0;
  cursor: pointer;
  font-weight: bold;
}
.advanced-content {
  margin-top: 1rem;
  background: rgba(0,0,0,0.2);
  padding: 1rem;
  border-radius: 8px;
}

.switch-row {
  display: flex;
  align-items: flex-start;
  gap: 0.8rem;
  margin-bottom: 1rem;
  cursor: pointer;
}
.switch-row input[type="checkbox"] {
  margin-top: 0.3rem;
  width: 16px;
  height: 16px;
}
.switch-title {
  color: var(--text-main);
  font-weight: bold;
  font-size: 0.95rem;
}
.switch-desc {
  color: var(--text-muted);
  font-size: 0.8rem;
  margin-top: 0.2rem;
}

.modal-footer {
  margin-top: 2rem;
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
}
button {
  padding: 0.6rem 1.2rem;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  font-weight: bold;
  transition: opacity 0.2s;
}
button:hover { opacity: 0.8; }

.btn-cancel {
  background: transparent;
  color: var(--text-muted);
}
.btn-delete {
  background: rgba(239, 68, 68, 0.1);
  color: var(--accent-red);
}
.btn-save {
  background: linear-gradient(90deg, var(--accent-blue), var(--accent-cyan));
  color: white;
}

.btn-test {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  color: var(--text-main);
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-weight: bold;
  white-space: nowrap;
  transition: all 0.2s;
}
.btn-test:hover:not(:disabled) {
  border-color: var(--accent-green);
  color: var(--accent-green);
}
.btn-test:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ping-result {
  font-size: 12px;
  margin-top: 6px;
  font-weight: 600;
}
.ping-result.success { color: var(--accent-green); }
.ping-result.error { color: var(--accent-red); }
</style>
