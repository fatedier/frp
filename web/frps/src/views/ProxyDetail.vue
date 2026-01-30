<template>
  <div class="proxy-detail-page">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
      <a class="breadcrumb-link" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </a>
      <template v-if="fromClient">
        <router-link to="/clients" class="breadcrumb-item">Clients</router-link>
        <span class="breadcrumb-separator">/</span>
        <router-link :to="`/clients/${fromClient}`" class="breadcrumb-item">{{
          fromClient
        }}</router-link>
        <span class="breadcrumb-separator">/</span>
      </template>
      <template v-else>
        <router-link to="/proxies" class="breadcrumb-item">Proxies</router-link>
        <span class="breadcrumb-separator">/</span>
        <router-link
          v-if="proxy?.clientID"
          :to="clientLink"
          class="breadcrumb-item"
        >
          {{ proxy.user ? `${proxy.user}.${proxy.clientID}` : proxy.clientID }}
        </router-link>
        <span v-if="proxy?.clientID" class="breadcrumb-separator">/</span>
      </template>
      <span class="breadcrumb-current">{{ proxyName }}</span>
    </nav>

    <div v-loading="loading" class="detail-content">
      <template v-if="proxy">
        <!-- Header Section -->
        <div class="header-section">
          <div class="header-main">
            <div
              class="proxy-icon"
              :style="{ background: proxyIconConfig.gradient }"
            >
              <el-icon><component :is="proxyIconConfig.icon" /></el-icon>
            </div>
            <div class="header-info">
              <div class="header-title-row">
                <h1 class="proxy-name">{{ proxy.name }}</h1>
                <span class="type-tag">{{ proxy.type.toUpperCase() }}</span>
                <span class="status-badge" :class="proxy.status">
                  {{ proxy.status }}
                </span>
              </div>
              <div class="header-meta">
                <router-link
                  v-if="proxy.clientID"
                  :to="clientLink"
                  class="client-link"
                >
                  <el-icon><Monitor /></el-icon>
                  <span
                    >Client:
                    {{
                      proxy.user
                        ? `${proxy.user}.${proxy.clientID}`
                        : proxy.clientID
                    }}</span
                  >
                </router-link>
              </div>
            </div>
          </div>
        </div>

        <!-- Stats Cards -->
        <div class="stats-grid">
          <div v-if="proxy.port" class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Port</span>
              <div class="stat-icon port">
                <el-icon><Connection /></el-icon>
              </div>
            </div>
            <div class="stat-value">{{ proxy.port }}</div>
          </div>
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Connections</span>
              <div class="stat-icon connections">
                <el-icon><DataLine /></el-icon>
              </div>
            </div>
            <div class="stat-value">{{ proxy.conns }}</div>
          </div>
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Traffic In</span>
              <div class="stat-icon traffic-in">
                <el-icon><Bottom /></el-icon>
              </div>
            </div>
            <div class="stat-value">
              <span class="value-number">{{
                formatTrafficValue(proxy.trafficIn)
              }}</span>
              <span class="value-unit">{{
                formatTrafficUnit(proxy.trafficIn)
              }}</span>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-label">Traffic Out</span>
              <div class="stat-icon traffic-out">
                <el-icon><Top /></el-icon>
              </div>
            </div>
            <div class="stat-value">
              <span class="value-number">{{
                formatTrafficValue(proxy.trafficOut)
              }}</span>
              <span class="value-unit">{{
                formatTrafficUnit(proxy.trafficOut)
              }}</span>
            </div>
          </div>
        </div>

        <!-- Status Timeline -->
        <div class="timeline-card">
          <div class="timeline-header">
            <el-icon><DataLine /></el-icon>
            <h2>Status Timeline</h2>
          </div>
          <div class="timeline-body">
            <div class="timeline-grid">
              <div class="timeline-item">
                <span class="timeline-label">Last Start Time</span>
                <span class="timeline-value">{{
                  proxy.lastStartTime || '-'
                }}</span>
              </div>
              <div class="timeline-item">
                <span class="timeline-label">Last Close Time</span>
                <span class="timeline-value">{{
                  proxy.lastCloseTime || '-'
                }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Configuration Section -->
        <div class="config-section">
          <div class="config-section-header">
            <el-icon><Setting /></el-icon>
            <h2>Configuration</h2>
          </div>

          <!-- Config Cards Grid -->
          <div class="config-grid">
            <div class="config-item-card">
              <div class="config-item-icon encryption">
                <el-icon><Lock /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Encryption</span>
                <span class="config-item-value">{{
                  proxy.encryption ? 'Enabled' : 'Disabled'
                }}</span>
              </div>
            </div>

            <div class="config-item-card">
              <div class="config-item-icon compression">
                <el-icon><Lightning /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Compression</span>
                <span class="config-item-value">{{
                  proxy.compression ? 'Enabled' : 'Disabled'
                }}</span>
              </div>
            </div>

            <div v-if="proxy.customDomains" class="config-item-card">
              <div class="config-item-icon domains">
                <el-icon><Link /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Custom Domains</span>
                <span class="config-item-value">{{ proxy.customDomains }}</span>
              </div>
            </div>

            <div v-if="proxy.subdomain" class="config-item-card">
              <div class="config-item-icon subdomain">
                <el-icon><Link /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Subdomain</span>
                <span class="config-item-value">{{ proxy.subdomain }}</span>
              </div>
            </div>

            <div v-if="proxy.locations" class="config-item-card">
              <div class="config-item-icon locations">
                <el-icon><Location /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Locations</span>
                <span class="config-item-value">{{ proxy.locations }}</span>
              </div>
            </div>

            <div v-if="proxy.hostHeaderRewrite" class="config-item-card">
              <div class="config-item-icon host">
                <el-icon><Tickets /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Host Rewrite</span>
                <span class="config-item-value">{{
                  proxy.hostHeaderRewrite
                }}</span>
              </div>
            </div>

            <div v-if="proxy.multiplexer" class="config-item-card">
              <div class="config-item-icon multiplexer">
                <el-icon><Cpu /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Multiplexer</span>
                <span class="config-item-value">{{ proxy.multiplexer }}</span>
              </div>
            </div>

            <div v-if="proxy.routeByHTTPUser" class="config-item-card">
              <div class="config-item-icon route">
                <el-icon><Connection /></el-icon>
              </div>
              <div class="config-item-content">
                <span class="config-item-label">Route By HTTP User</span>
                <span class="config-item-value">{{
                  proxy.routeByHTTPUser
                }}</span>
              </div>
            </div>
          </div>

          <!-- Annotations -->
          <template v-if="proxy.annotations && proxy.annotations.size > 0">
            <div class="annotations-section">
              <div
                v-for="[key, value] in proxy.annotations"
                :key="key"
                class="annotation-tag"
              >
                {{ key }}: {{ value }}
              </div>
            </div>
          </template>
        </div>

        <!-- Traffic Card -->
        <div class="traffic-card">
          <div class="traffic-header">
            <h2>Traffic Statistics</h2>
          </div>
          <div class="traffic-body">
            <Traffic :proxy-name="proxyName" />
          </div>
        </div>
      </template>

      <div v-else-if="!loading" class="not-found">
        <h2>Proxy not found</h2>
        <p>The proxy doesn't exist or has been removed.</p>
        <router-link to="/proxies">
          <el-button type="primary">Back to Proxies</el-button>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowLeft,
  Monitor,
  Connection,
  DataLine,
  Bottom,
  Top,
  Link,
  Lock,
  Promotion,
  Grid,
  Setting,
  Cpu,
  Lightning,
  Tickets,
  Location,
} from '@element-plus/icons-vue'
import { getProxyByName } from '../api/proxy'
import { getServerInfo } from '../api/server'
import {
  BaseProxy,
  TCPProxy,
  UDPProxy,
  HTTPProxy,
  HTTPSProxy,
  TCPMuxProxy,
  STCPProxy,
  SUDPProxy,
} from '../utils/proxy'
import Traffic from '../components/Traffic.vue'

const route = useRoute()
const router = useRouter()
const proxyName = computed(() => route.params.name as string)
const fromClient = computed(() => {
  if (route.query.from === 'client' && route.query.client) {
    return route.query.client as string
  }
  return null
})
const proxy = ref<BaseProxy | null>(null)
const loading = ref(true)

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/proxies')
  }
}

let serverInfo: {
  vhostHTTPPort: number
  vhostHTTPSPort: number
  tcpmuxHTTPConnectPort: number
  subdomainHost: string
} | null = null

const clientLink = computed(() => {
  if (!proxy.value) return ''
  const key = proxy.value.user
    ? `${proxy.value.user}.${proxy.value.clientID}`
    : proxy.value.clientID
  return `/clients/${key}`
})

const proxyIconConfig = computed(() => {
  const type = proxy.value?.type?.toLowerCase() || ''
  const configs: Record<string, { icon: any; gradient: string }> = {
    tcp: {
      icon: Connection,
      gradient: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
    },
    udp: {
      icon: Promotion,
      gradient: 'linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%)',
    },
    http: {
      icon: Link,
      gradient: 'linear-gradient(135deg, #22c55e 0%, #16a34a 100%)',
    },
    https: {
      icon: Lock,
      gradient: 'linear-gradient(135deg, #14b8a6 0%, #0d9488 100%)',
    },
    stcp: {
      icon: Lock,
      gradient: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)',
    },
    sudp: {
      icon: Lock,
      gradient: 'linear-gradient(135deg, #f97316 0%, #ea580c 100%)',
    },
    tcpmux: {
      icon: Grid,
      gradient: 'linear-gradient(135deg, #06b6d4 0%, #0891b2 100%)',
    },
    xtcp: {
      icon: Connection,
      gradient: 'linear-gradient(135deg, #ec4899 0%, #db2777 100%)',
    },
  }
  return (
    configs[type] || {
      icon: Connection,
      gradient: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
    }
  )
})

const formatTrafficValue = (bytes: number): string => {
  if (bytes === 0) return '0'
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)
  return value < 10 ? value.toFixed(1) : Math.round(value).toString()
}

const formatTrafficUnit = (bytes: number): string => {
  if (bytes === 0) return 'B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return units[i]
}

const fetchServerInfo = async () => {
  if (serverInfo) return serverInfo
  const res = await getServerInfo()
  serverInfo = res
  return serverInfo
}

const fetchProxy = async () => {
  const name = proxyName.value
  if (!name) {
    loading.value = false
    return
  }

  try {
    const data = await getProxyByName(name)
    const info = await fetchServerInfo()
    const type = data.conf?.type || ''

    if (type === 'tcp') {
      proxy.value = new TCPProxy(data)
    } else if (type === 'udp') {
      proxy.value = new UDPProxy(data)
    } else if (type === 'http' && info?.vhostHTTPPort) {
      proxy.value = new HTTPProxy(data, info.vhostHTTPPort, info.subdomainHost)
    } else if (type === 'https' && info?.vhostHTTPSPort) {
      proxy.value = new HTTPSProxy(
        data,
        info.vhostHTTPSPort,
        info.subdomainHost,
      )
    } else if (type === 'tcpmux' && info?.tcpmuxHTTPConnectPort) {
      proxy.value = new TCPMuxProxy(
        data,
        info.tcpmuxHTTPConnectPort,
        info.subdomainHost,
      )
    } else if (type === 'stcp') {
      proxy.value = new STCPProxy(data)
    } else if (type === 'sudp') {
      proxy.value = new SUDPProxy(data)
    } else {
      proxy.value = new BaseProxy(data)
      proxy.value.type = type
    }
  } catch (error: any) {
    ElMessage.error('Failed to fetch proxy: ' + error.message)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProxy()
})
</script>

<style scoped>
.proxy-detail-page {
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  margin-bottom: 24px;
}

.breadcrumb-link {
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s;
  margin-right: 4px;
}

.breadcrumb-link:hover {
  color: var(--text-primary);
}

.breadcrumb-item {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s;
}

.breadcrumb-item:hover {
  color: var(--el-color-primary);
}

.breadcrumb-separator {
  color: var(--el-border-color);
}

.breadcrumb-current {
  color: var(--text-primary);
  font-weight: 500;
}

/* Header Section */
.header-section {
  margin-bottom: 24px;
}

.header-main {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.proxy-icon {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 26px;
  color: white;
}

.header-info {
  flex: 1;
  min-width: 0;
}

.header-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.proxy-name {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
  line-height: 1.3;
  word-break: break-all;
}

.type-tag {
  font-size: 12px;
  font-weight: 500;
  padding: 4px 12px;
  border-radius: 20px;
  background: var(--el-fill-color-dark);
  color: var(--el-text-color-secondary);
  border: 1px solid var(--el-border-color-lighter);
}

.status-badge {
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  text-transform: capitalize;
}

.status-badge.online {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}

.status-badge.offline {
  background: var(--hover-bg);
  color: var(--text-secondary);
}

html.dark .status-badge.online {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.header-meta {
  display: flex;
  align-items: center;
  gap: 16px;
}

.client-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s;
}

.client-link:hover {
  color: var(--el-color-primary);
}

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  padding: 20px;
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.stat-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.stat-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.stat-icon.port {
  background: rgba(139, 92, 246, 0.1);
  color: #8b5cf6;
}

.stat-icon.connections {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.stat-icon.traffic-in {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.stat-icon.traffic-out {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

html.dark .stat-icon.port {
  background: rgba(139, 92, 246, 0.15);
}

html.dark .stat-icon.connections {
  background: rgba(168, 85, 247, 0.15);
}

html.dark .stat-icon.traffic-in {
  background: rgba(59, 130, 246, 0.15);
}

html.dark .stat-icon.traffic-out {
  background: rgba(34, 197, 94, 0.15);
}

.stat-value {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.value-number {
  font-size: 28px;
  font-weight: 500;
  color: var(--text-primary);
  line-height: 1;
}

.stat-value:not(:has(.value-number)) {
  font-size: 28px;
  font-weight: 500;
  color: var(--text-primary);
}

.value-unit {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
}

/* Timeline Card */
.timeline-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  margin-bottom: 16px;
}

.timeline-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 20px;
  color: var(--text-secondary);
}

.timeline-header h2 {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.timeline-body {
  padding: 20px;
  padding-top: 0;
}

.timeline-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
  background: var(--el-fill-color-light);
  border-radius: 10px;
  padding: 20px 24px;
}

.timeline-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.timeline-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.timeline-value {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
}

/* Card Base */
.traffic-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  margin-bottom: 16px;
}

/* Config Section */
.config-section {
  margin-bottom: 24px;
}

.config-section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  color: var(--text-secondary);
}

.config-section-header h2 {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.config-item-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
}

.config-item-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}

.config-item-icon.encryption {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.config-item-icon.compression {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.config-item-icon.domains {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.config-item-icon.subdomain {
  background: rgba(168, 85, 247, 0.1);
  color: #a855f7;
}

.config-item-icon.locations {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.config-item-icon.host {
  background: rgba(249, 115, 22, 0.1);
  color: #f97316;
}

.config-item-icon.multiplexer {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.config-item-icon.route {
  background: rgba(236, 72, 153, 0.1);
  color: #ec4899;
}

html.dark .config-item-icon.encryption,
html.dark .config-item-icon.compression {
  background: rgba(34, 197, 94, 0.15);
}

html.dark .config-item-icon.domains,
html.dark .config-item-icon.subdomain {
  background: rgba(168, 85, 247, 0.15);
}

html.dark .config-item-icon.locations,
html.dark .config-item-icon.multiplexer {
  background: rgba(59, 130, 246, 0.15);
}

html.dark .config-item-icon.host {
  background: rgba(249, 115, 22, 0.15);
}

html.dark .config-item-icon.route {
  background: rgba(236, 72, 153, 0.15);
}

.config-item-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.config-item-label {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.config-item-value {
  font-size: 15px;
  color: var(--text-primary);
  font-weight: 500;
  word-break: break-all;
}

.annotations-section {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.annotation-tag {
  display: inline-flex;
  padding: 6px 12px;
  background: var(--el-fill-color);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.traffic-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--header-border);
}

.traffic-header h2 {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

/* Traffic Card */
.traffic-body {
  padding: 20px;
}

/* Not Found */
.not-found {
  text-align: center;
  padding: 60px 20px;
}

.not-found h2 {
  font-size: 18px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.not-found p {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 20px;
}

/* Responsive */
@media (max-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .header-main {
    flex-direction: column;
    gap: 16px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .timeline-grid {
    grid-template-columns: 1fr;
  }
}
</style>
