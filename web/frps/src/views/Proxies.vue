<template>
  <div class="proxies-page">
    <div class="page-header">
      <div class="header-top">
        <div class="title-section">
          <h1 class="page-title">Proxies</h1>
          <p class="page-subtitle">View and manage all proxy configurations</p>
        </div>

        <div class="actions-section">
          <ActionButton variant="outline" size="small" @click="refreshData">
            Refresh
          </ActionButton>

          <ActionButton variant="outline" size="small" danger @click="showClearDialog = true">
            Clear Offline
          </ActionButton>
        </div>
      </div>

      <div class="filter-section">
        <div class="search-row">
          <el-input
            v-model="searchText"
            placeholder="Search proxies..."
            :prefix-icon="Search"
            clearable
            class="main-search"
          />
        </div>

        <div class="type-tabs">
          <button
            v-for="t in proxyTypes"
            :key="t.value"
            class="type-tab"
            :class="{ active: activeType === t.value }"
            @click="activeType = t.value"
          >
            {{ t.label }}
          </button>
        </div>
      </div>
    </div>

    <div v-loading="loading" class="proxies-content">
      <div v-if="proxies.length > 0" class="proxies-list">
        <ProxyCard
          v-for="proxy in proxies"
          :key="`${proxy.type}:${proxy.name}`"
          :proxy="proxy"
          :show-type="activeType === 'all'"
        />
      </div>
      <div v-else-if="!loading" class="empty-state">
        <el-empty description="No proxies found" />
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

    <ConfirmDialog
      v-model="showClearDialog"
      title="Clear Offline"
      message="Are you sure you want to clear all offline proxies?"
      confirm-text="Clear"
      danger
      @confirm="handleClearConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElPagination } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import ConfirmDialog from '@shared/components/ConfirmDialog.vue'
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
import ProxyCard from '../components/ProxyCard.vue'
import {
  getProxiesV2,
  clearOfflineProxies as apiClearOfflineProxies,
} from '../api/proxy'
import { getServerInfo } from '../api/server'
import type { ProxyStatsInfo } from '../types/proxy'
import type { ServerInfo } from '../types/server'

const route = useRoute()
const router = useRouter()

const proxyTypes = [
  { label: 'All', value: 'all' },
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'https' },
  { label: 'TCPMUX', value: 'tcpmux' },
  { label: 'STCP', value: 'stcp' },
  { label: 'XTCP', value: 'xtcp' },
  { label: 'SUDP', value: 'sudp' },
]

const activeType = ref((route.params.type as string) || 'all')
const proxies = ref<BaseProxy[]>([])
const loading = ref(false)
const searchText = ref('')
const showClearDialog = ref(false)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
let requestSeq = 0
let searchDebounceTimer: number | null = null

// Server info cache - cache the Promise itself so concurrent first calls
// from Promise.all (convertProxies) don't kick off multiple HTTP requests.
let serverInfoPromise: Promise<ServerInfo> | null = null

const fetchServerInfo = (): Promise<ServerInfo> => {
  if (!serverInfoPromise) {
    serverInfoPromise = getServerInfo().catch((err) => {
      // Allow retry after failure
      serverInfoPromise = null
      throw err
    })
  }
  return serverInfoPromise
}

const convertProxy = async (
  proxy: ProxyStatsInfo,
): Promise<BaseProxy | null> => {
  const type = proxy.type || activeType.value
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
  // Fallback for types without a dedicated class (e.g. xtcp). Matches the
  // pattern in ProxyDetail.vue so the type tag and meta render correctly.
  const bp = new BaseProxy(proxy)
  bp.type = type
  return bp
}

const convertProxies = async (items: ProxyStatsInfo[]): Promise<BaseProxy[]> => {
  const converted = await Promise.all(items.map((item) => convertProxy(item)))
  return converted.filter((item): item is BaseProxy => item !== null)
}

const fetchData = async (silent = false) => {
  const seq = ++requestSeq
  if (!silent) loading.value = true

  try {
    const q = searchText.value.trim()
    const data = await getProxiesV2({
      page: page.value,
      pageSize: pageSize.value,
      type: activeType.value === 'all' ? undefined : activeType.value,
      q: q || undefined,
    })
    if (seq !== requestSeq) return

    const maxPage = Math.max(1, Math.ceil(data.total / data.pageSize))
    if (data.items.length === 0 && data.total > 0 && data.page > maxPage) {
      page.value = maxPage
      await fetchData(silent)
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
    ElMessage({
      showClose: true,
      message: 'Failed to fetch proxies: ' + error.message,
      type: 'error',
    })
  } finally {
    if (seq === requestSeq) {
      loading.value = false
    }
  }
}

const clearSearchDebounce = () => {
  if (searchDebounceTimer !== null) {
    window.clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
}

const resetPageAndFetch = () => {
  clearSearchDebounce()
  page.value = 1
  fetchData()
}

const refreshData = () => {
  fetchData()
}

const onPageChange = (value: number) => {
  clearSearchDebounce()
  page.value = value
  fetchData()
}

const onPageSizeChange = (value: number) => {
  pageSize.value = value
  resetPageAndFetch()
}

const handleClearConfirm = async () => {
  showClearDialog.value = false
  await clearOfflineProxies()
}

const clearOfflineProxies = async () => {
  try {
    await apiClearOfflineProxies()
    ElMessage({
      message: 'Successfully cleared offline proxies',
      type: 'success',
    })
    fetchData()
  } catch (err: any) {
    ElMessage({
      message: 'Failed to clear offline proxies: ' + err.message,
      type: 'warning',
    })
  }
}

const sanitizeClientQuery = () => {
  const hasClientQuery =
    Object.prototype.hasOwnProperty.call(route.query, 'clientID') ||
    Object.prototype.hasOwnProperty.call(route.query, 'user')
  if (!hasClientQuery) return

  const query = { ...route.query }
  delete query.clientID
  delete query.user
  router.replace({ query })
}

// Watch for type changes
watch(activeType, (newType) => {
  clearSearchDebounce()
  page.value = 1
  // Update route but preserve query params
  router.replace({ params: { type: newType }, query: route.query })
  fetchData()
})

watch(searchText, () => {
  clearSearchDebounce()
  page.value = 1
  searchDebounceTimer = window.setTimeout(() => {
    searchDebounceTimer = null
    fetchData()
  }, 300)
})

watch(() => route.query, sanitizeClientQuery)

onUnmounted(() => {
  clearSearchDebounce()
})

// Initial fetch
sanitizeClientQuery()
fetchData()
</script>

<style scoped>
.proxies-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.title-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.page-title {
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0;
  line-height: 1.2;
}

.page-subtitle {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin: 0;
}

.actions-section {
  display: flex;
  gap: 12px;
}


.filter-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-top: 8px;
}

.search-row {
  display: flex;
  gap: 16px;
  width: 100%;
  align-items: center;
}

.main-search {
  flex: 1;
}

.main-search :deep(.el-input__wrapper) {
  height: 32px;
  border-radius: 8px;
}

.type-tabs {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
}

.type-tab {
  padding: 6px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  text-transform: uppercase;
}

.type-tab:hover {
  background: var(--el-fill-color-light);
}

.type-tab.active {
  background: var(--el-fill-color-darker);
  color: var(--el-text-color-primary);
  border-color: var(--el-fill-color-darker);
}

.proxies-content {
  min-height: 200px;
}

.proxies-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.empty-state {
  padding: 60px 0;
}

.pagination-section {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  .search-row {
    flex-direction: column;
  }

  .pagination-section {
    justify-content: center;
  }
}
</style>
