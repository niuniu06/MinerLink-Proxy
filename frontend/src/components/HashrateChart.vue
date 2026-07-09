<template>
  <div class="hashrate-chart-container glass-panel">
    <div class="chart-header">
      <h3 class="chart-title">
        <svg class="chart-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12V7a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h7m4-10-6 6-4-4"></path></svg>
        {{ title }}
      </h3>
    </div>
    
    <!-- Numerical Hashrate Blocks -->
    <div class="stats-blocks">
      <div class="stat-block">
        <div class="stat-value">{{ formatHashrateString(avg10m) }}</div>
        <div class="stat-label">10分钟算力</div>
      </div>
      <div class="stat-block">
        <div class="stat-value">{{ formatHashrateString(avg1h) }}</div>
        <div class="stat-label">1小时算力</div>
      </div>
      <div class="stat-block">
        <div class="stat-value">{{ formatHashrateString(avg6h) }}</div>
        <div class="stat-label">6小时算力</div>
      </div>
    </div>

    <!-- 3-Day Curve -->
    <div class="chart-body">
      <v-chart class="chart" :option="chartOption" autoresize />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, DataZoomComponent } from 'echarts/components'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent
])

const props = defineProps({
  historyData: {
    type: Object,
    default: () => ({ summary: {}, history: [] })
  },
  title: {
    type: String,
    default: '算力曲线'
  }
})

const formatHashrateString = (value) => {
  if (!value) return '0.00 H/s'
  const val = value * 1e6;
  if (val >= 1e15) return (val / 1e15).toFixed(2) + ' PH/s'
  if (val >= 1e12) return (val / 1e12).toFixed(2) + ' TH/s'
  if (val >= 1e9) return (val / 1e9).toFixed(2) + ' GH/s'
  if (val >= 1e6) return (val / 1e6).toFixed(2) + ' MH/s'
  if (val >= 1e3) return (val / 1e3).toFixed(2) + ' KH/s'
  return val.toFixed(2) + ' H/s'
}

// Use backend provided summary data
const avg10m = computed(() => {
  const s = props.historyData?.summary?.avg10m;
  if (!s) return 0;
  return s.mainHashrate + s.feeHashrate;
});

const avg1h = computed(() => {
  const s = props.historyData?.summary?.avg1h;
  if (!s) return 0;
  return s.mainHashrate + s.feeHashrate;
});

const avg6h = computed(() => {
  const s = props.historyData?.summary?.avg6h;
  if (!s) return 0;
  return s.mainHashrate + s.feeHashrate;
});

const chartOption = computed(() => {
  const data = props.historyData?.history || [];
  const timestamps = data.map(d => {
    const dObj = new Date(d.timestamp)
    const year = dObj.getFullYear();
    const month = (dObj.getMonth() + 1).toString().padStart(2, '0');
    const day = dObj.getDate().toString().padStart(2, '0');
    const hours = dObj.getHours().toString().padStart(2, '0');
    return `${year}/${month}/${day} ${hours}:00`
  })
  const mainHash = data.map(d => d.mainHashrate)
  const feeHash = data.map(d => d.feeHashrate)
  
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(25, 29, 36, 0.9)',
      borderColor: 'rgba(99, 102, 241, 0.3)',
      textStyle: { color: '#e2e8f0' },
      valueFormatter: (value) => formatHashrateString(value)
    },
    legend: {
      data: ['矿机主算力', '抽水算力'],
      textStyle: { color: '#94a3b8' },
      top: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '12%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timestamps,
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { 
        color: '#94a3b8',
        formatter: function (value) {
          // value is "YYYY/MM/DD HH:00"
          // Display only HH:00 on the X-axis for a cleaner look
          const parts = value.split(' ');
          return parts[1];
        }
      }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: '#1e293b', type: 'dashed' } },
      axisLabel: { 
        color: '#94a3b8',
        formatter: (value) => {
            const val = value * 1e6;
            if (val >= 1e15) return (val/1e15).toFixed(1) + ' P'
            if (val >= 1e12) return (val/1e12).toFixed(1) + ' T'
            if (val >= 1e9) return (val/1e9).toFixed(1) + ' G'
            if (val >= 1e6) return (val/1e6).toFixed(1) + ' M'
            if (val >= 1e3) return (val/1e3).toFixed(1) + ' K'
            return val
        }
      }
    },
    series: [
      {
        name: '矿机主算力',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { color: '#6366f1', width: 2 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(99, 102, 241, 0.4)' },
              { offset: 1, color: 'rgba(99, 102, 241, 0.0)' }
            ]
          }
        },
        data: mainHash
      },
      {
        name: '抽水算力',
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { color: '#ec4899', width: 2 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(236, 72, 153, 0.3)' },
              { offset: 1, color: 'rgba(236, 72, 153, 0.0)' }
            ]
          }
        },
        data: feeHash
      }
    ]
  }
})
</script>

<style scoped>
.hashrate-chart-container {
  width: 100%;
  padding: 1.5rem;
  border-radius: 16px;
  margin-bottom: 2rem;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.chart-header {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  margin-bottom: 1.5rem;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.25rem;
  font-weight: 600;
  color: #f8fafc;
  margin: 0;
}

.chart-icon {
  width: 20px;
  height: 20px;
  color: #6366f1;
}

.stats-blocks {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-block {
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 12px;
  padding: 1rem 1.5rem;
  min-width: 160px;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #f8fafc;
  margin-bottom: 0.25rem;
}

.stat-label {
  font-size: 0.85rem;
  color: #94a3b8;
}

.chart-body {
  height: 280px;
  width: 100%;
}

.chart {
  height: 100%;
  width: 100%;
}
</style>
