<template>
  <div class="traffic-chart-container" v-loading="loading">
    <div v-if="!loading && chartData.length > 0" class="chart-wrapper">
      <div class="y-axis">
        <div class="y-label">{{ formatFileSize(maxVal) }}</div>
        <div class="y-label">{{ formatFileSize(maxVal / 2) }}</div>
        <div class="y-label">0</div>
      </div>

      <div class="bars-area">
        <!-- Grid Lines -->
        <div class="grid-line top"></div>
        <div class="grid-line middle"></div>
        <div class="grid-line bottom"></div>

        <div v-for="(item, index) in chartData" :key="index" class="day-column">
          <div class="bars-group">
            <el-tooltip
              :content="`In: ${formatFileSize(item.in)}`"
              placement="top"
            >
              <div
                class="bar bar-in"
                :style="{ height: Math.max(item.inPercent, 1) + '%' }"
              ></div>
            </el-tooltip>
            <el-tooltip
              :content="`Out: ${formatFileSize(item.out)}`"
              placement="top"
            >
              <div
                class="bar bar-out"
                :style="{ height: Math.max(item.outPercent, 1) + '%' }"
              ></div>
            </el-tooltip>
          </div>
          <div class="date-label">{{ item.date }}</div>
        </div>
      </div>
    </div>

    <!-- Legend -->
    <div v-if="!loading && chartData.length > 0" class="legend">
      <div class="legend-item"><span class="dot in"></span> Traffic In</div>
      <div class="legend-item"><span class="dot out"></span> Traffic Out</div>
    </div>

    <el-empty v-else-if="!loading" description="No traffic data" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { formatFileSize } from '../utils/format'
import { getProxyTraffic } from '../api/proxy'
import type { TrafficResponse } from '../types/proxy'

const props = defineProps<{
  proxyName: string
}>()

const loading = ref(false)
const chartData = ref<
  Array<{
    date: string
    in: number
    out: number
    inPercent: number
    outPercent: number
  }>
>([])
const maxVal = ref(0)

const formatDateLabel = (date: string) => {
  const parts = date.split('-')
  if (parts.length !== 3) return date
  return `${Number(parts[1])}-${Number(parts[2])}`
}

const processData = (history: TrafficResponse['history'] = []) => {
  const points = history || []
  const maxIn = Math.max(0, ...points.map((item) => item.trafficIn))
  const maxOut = Math.max(0, ...points.map((item) => item.trafficOut))
  maxVal.value = Math.max(maxIn, maxOut, 100) // Minimum scale 100 bytes

  chartData.value = points.map((item) => ({
    date: formatDateLabel(item.date),
    in: item.trafficIn,
    out: item.trafficOut,
    inPercent: (item.trafficIn / maxVal.value) * 100,
    outPercent: (item.trafficOut / maxVal.value) * 100,
  }))
}

const fetchData = () => {
  loading.value = true
  getProxyTraffic(props.proxyName)
    .then((json) => {
      processData(json.history)
    })
    .catch((err) => {
      ElMessage({
        showClose: true,
        message: 'Get traffic info failed! ' + err,
        type: 'warning',
      })
    })
    .finally(() => {
      loading.value = false
    })
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.traffic-chart-container {
  width: 100%;
  height: 400px;
  display: flex;
  flex-direction: column;
  padding: 20px;
}

.chart-wrapper {
  flex: 1;
  display: flex;
  gap: 10px;
  position: relative;
  margin-bottom: 20px;
}

.y-axis {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  text-align: right;
  font-size: 12px;
  color: #909399;
  padding-bottom: 24px; /* Align with bars area excluding date labels */
  height: calc(100% - 24px); /* Subtract date label height approx */
}

.bars-area {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  position: relative;
  height: 100%;
  padding-bottom: 24px; /* Space for date labels */
}

.grid-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background-color: #e4e7ed;
  z-index: 0;
}

html.dark .grid-line {
  background-color: #3a3d5c;
}

.grid-line.top {
  top: 0;
}
.grid-line.middle {
  top: 50%;
  transform: translateY(-50%);
}
.grid-line.bottom {
  bottom: 24px;
} /* Align with bottom of bars */

.day-column {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  align-items: center;
  position: relative;
  z-index: 1;
}

.bars-group {
  height: 100%;
  display: flex;
  align-items: flex-end;
  gap: 4px;
  width: 60%;
}

.bar {
  flex: 1;
  border-radius: 4px 4px 0 0;
  transition: height 0.3s ease;
  min-height: 1px;
}

.bar-in {
  background-color: #5470c6;
}

.bar-out {
  background-color: #91cc75;
}

.bar:hover {
  opacity: 0.8;
}

.date-label {
  position: absolute;
  bottom: -24px;
  font-size: 12px;
  color: #909399;
  width: 100%;
  text-align: center;
}

.legend {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-top: 10px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
}

html.dark .legend-item {
  color: #e5e7eb;
}

.dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.dot.in {
  background-color: #5470c6;
}
.dot.out {
  background-color: #91cc75;
}
</style>
