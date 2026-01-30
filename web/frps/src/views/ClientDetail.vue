<template>
  <div class="client-detail-page">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
      <a class="breadcrumb-link" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </a>
      <router-link to="/clients" class="breadcrumb-item">Clients</router-link>
      <span class="breadcrumb-separator">/</span>
      <span class="breadcrumb-current">{{
        client?.displayName || route.params.key
      }}</span>
    </nav>

    <div v-loading="loading" class="detail-content">
      <template v-if="client">
        <!-- Header Card -->
        <div class="header-card">
          <div class="header-main">
            <div class="header-left">
              <div class="client-avatar">
                {{ client.displayName.charAt(0).toUpperCase() }}
              </div>
              <div class="client-info">
                <h1 class="client-name">{{ client.displayName }}</h1>
                <div class="client-meta">
                  <span v-if="client.ip" class="meta-item">{{
                    client.ip
                  }}</span>
                  <span v-if="client.hostname" class="meta-item">{{
                    client.hostname
                  }}</span>
                </div>
              </div>
            </div>
            <div class="header-right">
              <span
                class="status-badge"
                :class="client.online ? 'online' : 'offline'"
              >
                {{ client.online ? 'Online' : 'Offline' }}
              </span>
            </div>
          </div>

          <!-- Info Section -->
          <div class="info-section">
            <div class="info-item">
              <span class="info-label">Connections</span>
              <span class="info-value">{{ totalConnections }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">Run ID</span>
              <span class="info-value">{{ client.runID }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">First Connected</span>
              <span class="info-value">{{ client.firstConnectedAgo }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{
                client.online ? 'Connected' : 'Disconnected'
              }}</span>
              <span class="info-value">{{
                client.online ? client.lastConnectedAgo : client.disconnectedAgo
              }}</span>
            </div>
          </div>
        </div>

        <!-- Proxies Card -->
        <div class="proxies-card">
          <div class="proxies-header">
            <div class="proxies-title">
              <h2>Proxies</h2>
              <span class="proxies-count">{{ filteredProxies.length }}</span>
            </div>
            <el-input
              v-model="proxySearch"
              placeholder="Search proxies..."
              :prefix-icon="Search"
              clearable
              class="proxy-search"
            />
          </div>
          <div class="proxies-body">
            <div v-if="proxiesLoading" class="loading-state">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>Loading...</span>
            </div>
            <div v-else-if="filteredProxies.length > 0" class="proxies-list">
              <ProxyCard
                v-for="proxy in filteredProxies"
                :key="proxy.name"
                :proxy="proxy"
                show-type
              />
            </div>
            <div v-else-if="clientProxies.length > 0" class="empty-state">
              <p>No proxies match "{{ proxySearch }}"</p>
            </div>
            <div v-else class="empty-state">
              <p>No proxies found</p>
            </div>
          </div>
        </div>
      </template>

      <div v-else-if="!loading" class="not-found">
        <h2>Client not found</h2>
        <p>The client doesn't exist or has been removed.</p>
        <router-link to="/clients">
          <el-button type="primary">Back to Clients</el-button>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Loading, Search } from '@element-plus/icons-vue'
import { Client } from '../utils/client'
import { getClient } from '../api/client'
import { getProxiesByType } from '../api/proxy'
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
import { getServerInfo } from '../api/server'
import ProxyCard from '../components/ProxyCard.vue'

const route = useRoute()
const router = useRouter()
const client = ref<Client | null>(null)
const loading = ref(true)

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/clients')
  }
}
const proxiesLoading = ref(false)
const allProxies = ref<BaseProxy[]>([])
const proxySearch = ref('')

let serverInfo: {
  vhostHTTPPort: number
  vhostHTTPSPort: number
  tcpmuxHTTPConnectPort: number
  subdomainHost: string
} | null = null

const clientProxies = computed(() => {
  if (!client.value) return []
  return allProxies.value.filter(
    (p) =>
      p.clientID === client.value!.clientID && p.user === client.value!.user,
  )
})

const filteredProxies = computed(() => {
  if (!proxySearch.value) return clientProxies.value
  const search = proxySearch.value.toLowerCase()
  return clientProxies.value.filter(
    (p) =>
      p.name.toLowerCase().includes(search) ||
      p.type.toLowerCase().includes(search),
  )
})

const totalConnections = computed(() => {
  return clientProxies.value.reduce((sum, p) => sum + p.conns, 0)
})

const fetchServerInfo = async () => {
  if (serverInfo) return serverInfo
  const res = await getServerInfo()
  serverInfo = res
  return serverInfo
}

const fetchClient = async () => {
  const key = route.params.key as string
  if (!key) {
    loading.value = false
    return
  }
  try {
    const data = await getClient(key)
    client.value = new Client(data)
  } catch (error: any) {
    ElMessage.error('Failed to fetch client: ' + error.message)
  } finally {
    loading.value = false
  }
}

const fetchProxies = async () => {
  proxiesLoading.value = true
  const proxyTypes = ['tcp', 'udp', 'http', 'https', 'tcpmux', 'stcp', 'sudp']
  const proxies: BaseProxy[] = []
  try {
    const info = await fetchServerInfo()
    for (const type of proxyTypes) {
      try {
        const json = await getProxiesByType(type)
        if (!json.proxies) continue
        if (type === 'tcp') {
          proxies.push(...json.proxies.map((p: any) => new TCPProxy(p)))
        } else if (type === 'udp') {
          proxies.push(...json.proxies.map((p: any) => new UDPProxy(p)))
        } else if (type === 'http' && info?.vhostHTTPPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new HTTPProxy(p, info.vhostHTTPPort, info.subdomainHost),
            ),
          )
        } else if (type === 'https' && info?.vhostHTTPSPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new HTTPSProxy(p, info.vhostHTTPSPort, info.subdomainHost),
            ),
          )
        } else if (type === 'tcpmux' && info?.tcpmuxHTTPConnectPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new TCPMuxProxy(
                  p,
                  info.tcpmuxHTTPConnectPort,
                  info.subdomainHost,
                ),
            ),
          )
        } else if (type === 'stcp') {
          proxies.push(...json.proxies.map((p: any) => new STCPProxy(p)))
        } else if (type === 'sudp') {
          proxies.push(...json.proxies.map((p: any) => new SUDPProxy(p)))
        }
      } catch {
        // Ignore
      }
    }
    allProxies.value = proxies
  } catch {
    // Ignore
  } finally {
    proxiesLoading.value = false
  }
}

onMounted(() => {
  fetchClient()
  fetchProxies()
})
</script>

<style scoped>
.client-detail-page {
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

/* Card Base */
.header-card,
.proxies-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  margin-bottom: 16px;
}

/* Header Card */
.header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 24px;
}

.header-left {
  display: flex;
  gap: 16px;
  align-items: center;
}

.client-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 500;
  flex-shrink: 0;
}

.client-info {
  min-width: 0;
}

.client-name {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  line-height: 1.3;
}

.client-meta {
  display: flex;
  gap: 12px;
  font-size: 14px;
  color: var(--text-secondary);
}

.status-badge {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
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

/* Info Section */
.info-section {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  padding: 16px 24px;
}

.info-item {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.info-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.info-label::after {
  content: ':';
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 500;
  word-break: break-all;
}

/* Proxies Card */
.proxies-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  gap: 16px;
}

.proxies-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.proxies-title h2 {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.proxies-count {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--hover-bg);
  padding: 4px 10px;
  border-radius: 6px;
}

.proxy-search {
  width: 200px;
}

.proxy-search :deep(.el-input__wrapper) {
  border-radius: 6px;
}

.proxies-body {
  padding: 16px;
}

.proxies-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.empty-state p {
  margin: 0;
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
@media (max-width: 640px) {
  .header-main {
    flex-direction: column;
    gap: 16px;
  }

  .header-right {
    align-self: flex-start;
  }
}
</style>
