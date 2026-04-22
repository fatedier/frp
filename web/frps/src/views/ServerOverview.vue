<template>
  <div class="server-overview">
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Clients"
          :value="data.clientCounts"
          type="clients"
          subtitle="Connected clients"
          to="/clients"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Proxies"
          :value="data.proxyCounts"
          type="proxies"
          subtitle="Active proxies"
          to="/proxies/tcp"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Connections"
          :value="data.curConns"
          type="connections"
          subtitle="Current connections"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Traffic"
          :value="formatTrafficTotal()"
          type="traffic"
          subtitle="Total today"
        />
      </el-col>
    </el-row>

    <el-row :gutter="20" class="charts-row">
      <el-col :xs="24" :md="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Network Traffic</span>
              <el-tag size="small" type="info">Today</el-tag>
            </div>
          </template>
          <div class="traffic-summary">
            <div class="traffic-item in">
              <div class="traffic-icon">
                <el-icon><Download /></el-icon>
              </div>
              <div class="traffic-info">
                <div class="label">Inbound</div>
                <div class="value">
                  {{ formatFileSize(data.totalTrafficIn) }}
                </div>
              </div>
            </div>
            <div class="traffic-divider"></div>
            <div class="traffic-item out">
              <div class="traffic-icon">
                <el-icon><Upload /></el-icon>
              </div>
              <div class="traffic-info">
                <div class="label">Outbound</div>
                <div class="value">
                  {{ formatFileSize(data.totalTrafficOut) }}
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Proxy Types</span>
              <el-tag size="small" type="info">Now</el-tag>
            </div>
          </template>
          <div class="proxy-types-grid">
            <div
              v-for="(count, type) in data.proxyTypeCounts"
              :key="type"
              class="proxy-type-item"
              v-show="count > 0"
            >
              <div class="proxy-type-name">{{ type.toUpperCase() }}</div>
              <div class="proxy-type-count">{{ count }}</div>
            </div>
            <div v-if="!hasActiveProxies" class="no-data">
              No active proxies
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card v-if="data.autoTransportEnabled" class="config-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">Auto Transport</span>
          <el-tag size="small" type="success">{{ data.transportProtocol }}</el-tag>
        </div>
      </template>
      <div class="auto-grid">
        <div class="config-item">
          <span class="config-label">Advertised</span>
          <div class="tag-list">
            <el-tag
              v-for="protocol in data.autoTransportProtocols"
              :key="protocol"
              size="small"
              type="info"
            >
              {{ protocol }}
            </el-tag>
          </div>
        </div>
        <div class="config-item">
          <span class="config-label">Negotiations</span>
          <span class="config-value">
            {{ data.autoNegotiationSuccess }} / {{ data.autoNegotiationFailure }}
          </span>
        </div>
        <div class="config-item">
          <span class="config-label">Illegal Selections</span>
          <span class="config-value">{{ illegalSelectionTotal }}</span>
        </div>
        <div class="config-item">
          <span class="config-label">Switches</span>
          <span class="config-value">{{ switchTotal }}</span>
        </div>
      </div>
      <div class="auto-metrics">
        <div class="metric-block">
          <div class="metric-title">Selections</div>
          <div v-if="selectionRows.length > 0" class="metric-list">
            <div v-for="row in selectionRows" :key="row.key" class="metric-row">
              <span>{{ row.key }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
          <div v-else class="no-data compact">No selections</div>
        </div>
        <div class="metric-block">
          <div class="metric-title">Online Clients</div>
          <div v-if="onlineRows.length > 0" class="metric-list">
            <div v-for="row in onlineRows" :key="row.key" class="metric-row">
              <span>{{ row.key }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
          <div v-else class="no-data compact">No online auto clients</div>
        </div>
        <div class="metric-block">
          <div class="metric-title">Switch Paths</div>
          <div v-if="switchRows.length > 0" class="metric-list">
            <div v-for="row in switchRows" :key="row.key" class="metric-row">
              <span>{{ row.key }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
          <div v-else class="no-data compact">No switches</div>
        </div>
      </div>
    </el-card>

    <el-card class="config-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">Server Configuration</span>
          <el-tag size="small" type="success">v{{ data.version }}</el-tag>
        </div>
      </template>
      <div class="config-grid">
        <div class="config-item">
          <span class="config-label">Bind Port</span>
          <span class="config-value">{{ data.bindPort }}</span>
        </div>
        <div class="config-item" v-if="data.kcpBindPort != 0">
          <span class="config-label">KCP Port</span>
          <span class="config-value">{{ data.kcpBindPort }}</span>
        </div>
        <div class="config-item" v-if="data.quicBindPort != 0">
          <span class="config-label">QUIC Port</span>
          <span class="config-value">{{ data.quicBindPort }}</span>
        </div>
        <div class="config-item" v-if="data.vhostHTTPPort != 0">
          <span class="config-label">HTTP Port</span>
          <span class="config-value">{{ data.vhostHTTPPort }}</span>
        </div>
        <div class="config-item" v-if="data.vhostHTTPSPort != 0">
          <span class="config-label">HTTPS Port</span>
          <span class="config-value">{{ data.vhostHTTPSPort }}</span>
        </div>
        <div class="config-item" v-if="data.tcpmuxHTTPConnectPort != 0">
          <span class="config-label">TCPMux Port</span>
          <span class="config-value">{{ data.tcpmuxHTTPConnectPort }}</span>
        </div>
        <div class="config-item" v-if="data.subdomainHost != ''">
          <span class="config-label">Subdomain Host</span>
          <span class="config-value">{{ data.subdomainHost }}</span>
        </div>
        <div class="config-item">
          <span class="config-label">Max Pool Count</span>
          <span class="config-value">{{ data.maxPoolCount }}</span>
        </div>
        <div class="config-item">
          <span class="config-label">Max Ports/Client</span>
          <span class="config-value">{{ data.maxPortsPerClient }}</span>
        </div>
        <div class="config-item" v-if="data.allowPortsStr != ''">
          <span class="config-label">Allow Ports</span>
          <span class="config-value">{{ data.allowPortsStr }}</span>
        </div>
        <div class="config-item" v-if="data.tlsForce">
          <span class="config-label">TLS Force</span>
          <el-tag size="small" type="warning">Enabled</el-tag>
        </div>
        <div class="config-item">
          <span class="config-label">Heartbeat Timeout</span>
          <span class="config-value">{{ data.heartbeatTimeout }}s</span>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { formatFileSize } from '../utils/format'
import { Download, Upload } from '@element-plus/icons-vue'
import StatCard from '../components/StatCard.vue'
import { getServerInfo } from '../api/server'

const data = ref({
  version: '',
  bindPort: 0,
  kcpBindPort: 0,
  quicBindPort: 0,
  vhostHTTPPort: 0,
  vhostHTTPSPort: 0,
  tcpmuxHTTPConnectPort: 0,
  subdomainHost: '',
  maxPoolCount: 0,
  maxPortsPerClient: '',
  allowPortsStr: '',
  tlsForce: false,
  heartbeatTimeout: 0,
  transportProtocol: '',
  autoTransportEnabled: false,
  autoTransportProtocols: [] as string[],
  autoNegotiationSuccess: 0,
  autoNegotiationFailure: 0,
  autoTransportSelections: {} as Record<string, number>,
  autoTransportClientCounts: {} as Record<string, number>,
  autoTransportSwitchCounts: {} as Record<string, number>,
  autoTransportIllegalSelections: {} as Record<string, number>,
  clientCounts: 0,
  curConns: 0,
  proxyCounts: 0,
  totalTrafficIn: 0,
  totalTrafficOut: 0,
  proxyTypeCounts: {} as Record<string, number>,
})

const hasActiveProxies = computed(() => {
  return Object.values(data.value.proxyTypeCounts).some((c) => c > 0)
})

const sortedRows = (record: Record<string, number>) => {
  return Object.entries(record || {})
    .filter(([, value]) => value > 0)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, value]) => ({ key, value }))
}

const selectionRows = computed(() => sortedRows(data.value.autoTransportSelections))
const onlineRows = computed(() => sortedRows(data.value.autoTransportClientCounts))
const switchRows = computed(() => sortedRows(data.value.autoTransportSwitchCounts))

const illegalSelectionTotal = computed(() => {
  return Object.values(data.value.autoTransportIllegalSelections).reduce(
    (sum, value) => sum + value,
    0,
  )
})

const switchTotal = computed(() => {
  return Object.values(data.value.autoTransportSwitchCounts).reduce(
    (sum, value) => sum + value,
    0,
  )
})

const formatTrafficTotal = () => {
  const total = data.value.totalTrafficIn + data.value.totalTrafficOut
  return formatFileSize(total)
}

const fetchData = async () => {
  try {
    const json = await getServerInfo()
    data.value.version = json.version
    data.value.bindPort = json.bindPort
    data.value.kcpBindPort = json.kcpBindPort
    data.value.quicBindPort = json.quicBindPort
    data.value.vhostHTTPPort = json.vhostHTTPPort
    data.value.vhostHTTPSPort = json.vhostHTTPSPort
    data.value.tcpmuxHTTPConnectPort = json.tcpmuxHTTPConnectPort
    data.value.subdomainHost = json.subdomainHost
    data.value.maxPoolCount = json.maxPoolCount
    data.value.maxPortsPerClient = String(json.maxPortsPerClient)
    if (data.value.maxPortsPerClient == '0') {
      data.value.maxPortsPerClient = 'no limit'
    }
    data.value.allowPortsStr = json.allowPortsStr
    data.value.tlsForce = json.tlsForce
    data.value.heartbeatTimeout = json.heartbeatTimeout
    data.value.transportProtocol = json.transportProtocol
    data.value.autoTransportEnabled = json.autoTransportEnabled
    data.value.autoTransportProtocols = json.autoTransportProtocols || []
    data.value.autoNegotiationSuccess = json.autoNegotiationSuccess || 0
    data.value.autoNegotiationFailure = json.autoNegotiationFailure || 0
    data.value.autoTransportSelections = json.autoTransportSelections || {}
    data.value.autoTransportClientCounts = json.autoTransportClientCounts || {}
    data.value.autoTransportSwitchCounts = json.autoTransportSwitchCounts || {}
    data.value.autoTransportIllegalSelections =
      json.autoTransportIllegalSelections || {}
    data.value.clientCounts = json.clientCounts
    data.value.curConns = json.curConns
    data.value.totalTrafficIn = json.totalTrafficIn
    data.value.totalTrafficOut = json.totalTrafficOut
    data.value.proxyTypeCounts = json.proxyTypeCount || {}

    data.value.proxyCounts = 0
    if (json.proxyTypeCount != null) {
      Object.values(json.proxyTypeCount).forEach((count: any) => {
        data.value.proxyCounts += count || 0
      })
    }
  } catch {
    ElMessage({
      showClose: true,
      message: 'Get server info from frps failed!',
      type: 'error',
    })
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.server-overview {
  padding: 0;
}

.stats-row {
  margin-bottom: 20px;
}

.charts-row {
  margin-bottom: 20px;
}

.chart-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  height: 100%;
}

html.dark .chart-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.config-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  margin-bottom: 20px;
}

html.dark .config-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

html.dark .card-title {
  color: #e5e7eb;
}

.traffic-summary {
  display: flex;
  align-items: center;
  justify-content: space-around;
  min-height: 120px;
  padding: 10px 0;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 16px;
}

.traffic-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.traffic-item.in .traffic-icon {
  background: rgba(84, 112, 198, 0.1);
  color: #5470c6;
}

.traffic-item.out .traffic-icon {
  background: rgba(145, 204, 117, 0.1);
  color: #91cc75;
}

.traffic-info {
  display: flex;
  flex-direction: column;
}

.traffic-info .label {
  font-size: 14px;
  color: #909399;
}

.traffic-info .value {
  font-size: 24px;
  font-weight: 500;
  color: #303133;
}

html.dark .traffic-info .value {
  color: #e5e7eb;
}

.traffic-divider {
  width: 1px;
  height: 60px;
  background: #e4e7ed;
}

html.dark .traffic-divider {
  background: #3a3d5c;
}

.proxy-types-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
  min-height: 120px;
  align-content: center;
  padding: 10px 0;
}

.proxy-type-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

html.dark .proxy-type-item {
  background: #1e1e2d;
}

.proxy-type-name {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
  margin-bottom: 4px;
}

.proxy-type-count {
  font-size: 20px;
  font-weight: 500;
  color: #303133;
}

html.dark .proxy-type-count {
  color: #e5e7eb;
}

.no-data {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #909399;
  font-size: 14px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

.auto-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.auto-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.metric-block {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 12px;
}

html.dark .metric-block {
  border-color: #3a3d5c;
}

.metric-title {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
  margin-bottom: 8px;
}

.metric-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 14px;
  color: #303133;
}

html.dark .metric-row {
  color: #e5e7eb;
}

.metric-row span {
  overflow-wrap: anywhere;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.compact {
  min-height: auto;
  height: auto;
  justify-content: flex-start;
  font-size: 13px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  transition: background 0.2s;
}

html.dark .config-item {
  background: #1e1e2d;
}

.config-label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

html.dark .config-label {
  color: #9ca3af;
}

.config-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
  word-break: break-all;
}

html.dark .config-value {
  color: #e5e7eb;
}

@media (max-width: 768px) {
  .chart-container {
    height: 250px;
  }

  .config-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
