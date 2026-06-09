<template>
  <div class="proxy-card">
    <div class="card-header">
      <div class="title-area">
        <h2>{{ config.coinName || 'Unknown' }}</h2>
        <span class="port-badge">PORT {{ config.listenPort }}</span>
        <button class="btn-edit" @click="$emit('edit')">⚙ 参数热修改</button>
      </div>
    </div>

    <div class="stats-grid">
      <!-- Box 1: Status -->
      <div class="stat-box">
        <div class="box-title">调度引擎状态</div>
        <div class="box-content status">
          <div class="status-indicator" :class="{ smooth: config.enableSmoothFee }">
            <span class="pulse"></span>
            {{ config.enableSmoothFee ? '平滑无感拦截中' : '静默拦截中' }}
            <span class="machine-count" v-if="stats">({{ stats.activeMiners }} 台)</span>
          </div>
          <div class="status-sub">
            {{ config.enableSmoothFee ? '全分布时间轮算法生效中' : '标准时间轮机制生效中' }}
            <span v-if="config.enableAsic" class="asic-tag"> | ASIC兼容开启</span>
          </div>
        </div>
      </div>

      <!-- Box 2: Miners -->
      <div class="stat-box">
        <div class="box-title">在线矿机</div>
        <div class="box-content">
          <div class="big-number">{{ stats ? stats.activeMiners : 0 }}</div>
          <div class="box-sub">并发连接数</div>
        </div>
      </div>

      <!-- Box 3: Fee Progress -->
      <div class="stat-box">
        <div class="box-title">抽水比例公开</div>
        <div class="box-content fee-content">
          <div class="fee-row">
            <span class="fee-author">作者: {{ config.devFeePercent }}%</span>
            <span class="fee-op">运营者: {{ config.operatorFeePercent || '0.0' }}%</span>
          </div>
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: totalFeePercent + '%' }"></div>
          </div>
          <div class="fee-total">总比例: {{ totalFeePercent.toFixed(1) }}%</div>
        </div>
      </div>

      <!-- Box 4: Shares -->
      <div class="stat-box">
        <div class="box-title">总提交份额</div>
        <div class="box-content">
          <div class="big-number">{{ stats ? stats.totalShares : 0 }}</div>
          <div class="box-sub fee-intercept">抽水拦截: {{ stats ? stats.totalFeeShares : 0 }}</div>
        </div>
      </div>
    </div>

    <MinerTable :port="config.listenPort" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import MinerTable from './MinerTable.vue'

const props = defineProps({
  config: Object,
  stats: Object
})

const totalFeePercent = computed(() => {
  return (props.config.devFeePercent || 0) + (props.config.operatorFeePercent || 0)
})
</script>

<style scoped>
.proxy-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  padding: 18px 24px;
  box-shadow: none;
  transition: border 0.2s;
}

.proxy-card:hover {
  border-color: var(--accent-blue);
}

.card-header {
  margin-bottom: 1.5rem;
  border-bottom: 1px solid rgba(255,255,255,0.05);
  padding-bottom: 1rem;
}

.title-area {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.title-area h2 {
  margin: 0;
  font-size: 1.4rem;
  font-weight: 600;
  color: var(--text-main);
}

.port-badge {
  background: rgba(6, 182, 212, 0.15);
  color: var(--accent-cyan);
  padding: 0.3rem 0.8rem;
  border-radius: 20px;
  font-size: 0.9rem;
  font-weight: bold;
  border: 1px solid rgba(6, 182, 212, 0.3);
}

.btn-edit {
  margin-left: auto;
  background: var(--bg-color);
  color: var(--text-main);
  border: 1px solid var(--card-border);
  padding: 4px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 600;
  transition: border 0.2s;
}

.btn-edit:hover {
  border-color: var(--accent-blue);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-box {
  background: var(--bg-color);
  border: 1px solid var(--card-border);
  border-radius: 6px;
  padding: 1rem;
}

.box-title {
  color: var(--text-muted);
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  margin-bottom: 0.8rem;
}

.big-number {
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-main);
  line-height: 1;
  margin-bottom: 0.5rem;
  font-family: 'SF Mono', 'Fira Code', Consolas, monospace;
}

.box-sub {
  font-size: 0.85rem;
  color: var(--text-muted);
}

/* Custom Box Styling */
.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--accent-green);
  font-size: 1.2rem;
  font-weight: bold;
  margin-bottom: 0.8rem;
}

.status-indicator.smooth {
  color: var(--accent-cyan);
}

.pulse {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 10px currentColor;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.8); }
  100% { opacity: 1; transform: scale(1); }
}

.machine-count {
  font-weight: normal;
  font-size: 1rem;
}

.status-sub {
  font-size: 0.85rem;
  color: #6b7280;
}
.asic-tag { color: var(--accent-blue); }

.fee-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
  margin-bottom: 0.5rem;
}
.fee-author { color: var(--accent-blue); }
.fee-op { color: var(--accent-green); }

.progress-bar {
  height: 6px;
  background: #374151;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 0.5rem;
}
.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-blue), var(--accent-cyan));
}
.fee-total {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.fee-intercept {
  color: var(--accent-blue);
}

@media (max-width: 1024px) {
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 640px) {
  .stats-grid { grid-template-columns: 1fr; }
  .title-area { flex-wrap: wrap; }
}
</style>
