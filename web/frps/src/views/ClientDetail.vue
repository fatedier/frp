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
                <div class="client-name-row">
                  <h1 class="client-name">{{ client.displayName }}</h1>
                  <el-tag v-if="client.version" size="small" type="success"
                    >v{{ client.version }}</el-tag
                  >
                  <el-tag v-if="client.wireProtocolLabel" size="small" type="info">
                    {{ client.wireProtocolLabel }}
                  </el-tag>
                </div>
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
              <span class="info-value">{{ client.status.curConns }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">Run ID</span>
              <span class="info-value">{{ client.runID }}</span>
            </div>
            <div v-if="client.wireProtocol" class="info-item">
              <span class="info-label">Protocol</span>
              <span class="info-value">{{ client.wireProtocol }}</span>
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
              <span class="proxies-count">{{ total }}</span>
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
            <div v-else-if="proxies.length > 0" class="proxies-list">
              <ProxyCard
                v-for="proxy in proxies"
                :key="`${proxy.type}:${proxy.name}`"
                :proxy="proxy"
                show-type
              />
            </div>
            <div v-else-if="proxySearch.trim()" class="empty-state">
              <p>No proxies match "{{ proxySearch }}"</p>
            </div>
            <div v-else class="empty-state">
              <p>No proxies found</p>
            </div>
          </div>
          <div v-if="total > 0" class="pagination-section">
            <ElPagination
              :current-page="page"
              :page-size="pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="total"
              layout="total, sizes, prev, pager, next"
              @current-change="onPageChange"
              @size-change="onPageSizeChange"
            />
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
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElPagination } from 'element-plus'
import { ArrowLeft, Loading, Search } from '@element-plus/icons-vue'
import { Client } from '../utils/client'
import { getClientV2 } from '../api/client'
import { getProxiesV2 } from '../api/proxy'
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
import type { ProxyStatsInfo } from '../types/proxy'
import type { ServerInfo } from '../types/server'

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
const proxies = ref<BaseProxy[]>([])
const proxySearch = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
let requestSeq = 0
let searchDebounceTimer: number | null = null

let serverInfoPromise: Promise<ServerInfo> | null = null

const fetchServerInfo = (): Promise<ServerInfo> => {
  if (!serverInfoPromise) {
    serverInfoPromise = getServerInfo().catch((err) => {
      serverInfoPromise = null
      throw err
    })
  }
  return serverInfoPromise
}

const fetchClient = async (): Promise<boolean> => {
  const key = route.params.key as string
  if (!key) {
    loading.value = false
    return false
  }
  try {
    const data = await getClientV2(key)
    client.value = new Client(data)
    return true
  } catch (error: any) {
    ElMessage.error('Failed to fetch client: ' + error.message)
    return false
  } finally {
    loading.value = false
  }
}

const convertProxy = async (
  proxy: ProxyStatsInfo,
): Promise<BaseProxy | null> => {
  const type = proxy.type || ''
  if (type === 'tcp') {
    return new TCPProxy(proxy)
  }
  if (type === 'udp') {
    return new UDPProxy(proxy)
  }
  if (type === 'http') {
    const info = await fetchServerInfo()
    if (info && info.config.vhostHTTPPort) {
      return new HTTPProxy(
        proxy,
        info.config.vhostHTTPPort,
        info.config.subdomainHost,
      )
    }
    return null
  }
  if (type === 'https') {
    const info = await fetchServerInfo()
    if (info && info.config.vhostHTTPSPort) {
      return new HTTPSProxy(
        proxy,
        info.config.vhostHTTPSPort,
        info.config.subdomainHost,
      )
    }
    return null
  }
  if (type === 'tcpmux') {
    const info = await fetchServerInfo()
    if (info && info.config.tcpmuxHTTPConnectPort) {
      return new TCPMuxProxy(
        proxy,
        info.config.tcpmuxHTTPConnectPort,
        info.config.subdomainHost,
      )
    }
    return null
  }
  if (type === 'stcp') {
    return new STCPProxy(proxy)
  }
  if (type === 'sudp') {
    return new SUDPProxy(proxy)
  }

  const bp = new BaseProxy(proxy)
  bp.type = type
  return bp
}

const convertProxies = async (items: ProxyStatsInfo[]): Promise<BaseProxy[]> => {
  const converted = await Promise.all(items.map((item) => convertProxy(item)))
  return converted.filter((item): item is BaseProxy => item !== null)
}

const fetchProxies = async () => {
  if (!client.value) return
  const seq = ++requestSeq
  proxiesLoading.value = true

  try {
    const q = proxySearch.value.trim()
    const data = await getProxiesV2({
      page: page.value,
      pageSize: pageSize.value,
      q: q || undefined,
      clientID: client.value.clientID,
      user: client.value.user,
    })
    if (seq !== requestSeq) return

    const maxPage = Math.max(1, Math.ceil(data.total / data.pageSize))
    if (data.items.length === 0 && data.total > 0 && data.page > maxPage) {
      page.value = maxPage
      await fetchProxies()
      return
    }

    const converted = await convertProxies(data.items)
    if (seq !== requestSeq) return

    proxies.value = converted
    total.value = data.total
    page.value = data.page
    pageSize.value = data.pageSize
  } catch (error: any) {
    if (seq !== requestSeq) return
    ElMessage.error('Failed to fetch proxies: ' + error.message)
  } finally {
    if (seq === requestSeq) {
      proxiesLoading.value = false
    }
  }
}

const clearSearchDebounce = () => {
  if (searchDebounceTimer !== null) {
    window.clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
}

const invalidateProxyRequests = () => {
  requestSeq++
  proxiesLoading.value = false
}

const resetPageAndFetch = () => {
  clearSearchDebounce()
  page.value = 1
  fetchProxies()
}

const onPageChange = (value: number) => {
  clearSearchDebounce()
  page.value = value
  fetchProxies()
}

const onPageSizeChange = (value: number) => {
  pageSize.value = value
  resetPageAndFetch()
}

watch(proxySearch, () => {
  clearSearchDebounce()
  invalidateProxyRequests()
  page.value = 1
  searchDebounceTimer = window.setTimeout(() => {
    searchDebounceTimer = null
    fetchProxies()
  }, 300)
})

onUnmounted(() => {
  clearSearchDebounce()
})

onMounted(async () => {
  const ok = await fetchClient()
  if (!ok || !client.value) return

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

.client-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.client-name {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
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

.pagination-section {
  display: flex;
  justify-content: center;
  padding: 0 20px 20px;
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
