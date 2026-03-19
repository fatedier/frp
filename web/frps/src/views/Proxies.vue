<template>
  <div class="proxies-page">
    <div class="page-header">
      <div class="header-top">
        <div class="title-section">
          <h1 class="page-title">Proxies</h1>
          <p class="page-subtitle">View and manage all proxy configurations</p>
        </div>

        <div class="actions-section">
          <ActionButton variant="outline" size="small" @click="fetchData">
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

          <PopoverMenu
            :model-value="selectedClientKey"
            :width="220"
            placement="bottom-end"
            selectable
            filterable
            filter-placeholder="Search clients..."
            :display-value="selectedClientLabel"
            clearable
            class="client-filter"
            @update:model-value="onClientFilterChange($event as string)"
          >
            <template #default="{ filterText }">
              <PopoverMenuItem value="">All Clients</PopoverMenuItem>
              <PopoverMenuItem
                v-if="clientIDFilter && !selectedClientInList"
                :value="selectedClientKey"
              >
                {{ userFilter ? userFilter + '.' : '' }}{{ clientIDFilter }} (not found)
              </PopoverMenuItem>
              <PopoverMenuItem
                v-for="client in filteredClientOptions(filterText)"
                :key="client.key"
                :value="client.key"
              >
                {{ client.label }}
              </PopoverMenuItem>
            </template>
          </PopoverMenu>
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
      <div v-if="filteredProxies.length > 0" class="proxies-list">
        <ProxyCard
          v-for="proxy in filteredProxies"
          :key="proxy.name"
          :proxy="proxy"
        />
      </div>
      <div v-else-if="!loading" class="empty-state">
        <el-empty description="No proxies found" />
      </div>
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
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
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
import PopoverMenu from '@shared/components/PopoverMenu.vue'
import PopoverMenuItem from '@shared/components/PopoverMenuItem.vue'
import {
  getProxiesByType,
  clearOfflineProxies as apiClearOfflineProxies,
} from '../api/proxy'
import { getServerInfo } from '../api/server'
import { getClients } from '../api/client'
import { Client } from '../utils/client'

const route = useRoute()
const router = useRouter()

const proxyTypes = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'https' },
  { label: 'TCPMUX', value: 'tcpmux' },
  { label: 'STCP', value: 'stcp' },
  { label: 'SUDP', value: 'sudp' },
]

const activeType = ref((route.params.type as string) || 'tcp')
const proxies = ref<BaseProxy[]>([])
const clients = ref<Client[]>([])
const loading = ref(false)
const searchText = ref('')
const showClearDialog = ref(false)
const clientIDFilter = ref((route.query.clientID as string) || '')
const userFilter = ref((route.query.user as string) || '')

const clientOptions = computed(() => {
  return clients.value
    .map((c) => ({
      key: c.key,
      clientID: c.clientID,
      user: c.user,
      label: c.user ? `${c.user}.${c.clientID}` : c.clientID,
    }))
    .sort((a, b) => a.label.localeCompare(b.label))
})

// Compute selected client key for el-select v-model
const selectedClientKey = computed(() => {
  if (!clientIDFilter.value) return ''
  const client = clientOptions.value.find(
    (c) => c.clientID === clientIDFilter.value && c.user === userFilter.value,
  )
  // Return a synthetic key even if not found, so the select shows the filter is active
  return client?.key || `${userFilter.value}:${clientIDFilter.value}`
})

const selectedClientLabel = computed(() => {
  if (!clientIDFilter.value) return 'All Clients'
  const client = clientOptions.value.find(
    (c) => c.clientID === clientIDFilter.value && c.user === userFilter.value,
  )
  return client?.label || `${userFilter.value ? userFilter.value + '.' : ''}${clientIDFilter.value}`
})

const filteredClientOptions = (filterText: string) => {
  if (!filterText) return clientOptions.value
  const search = filterText.toLowerCase()
  return clientOptions.value.filter((c) => c.label.toLowerCase().includes(search))
}

// Check if the filtered client exists in the client list
const selectedClientInList = computed(() => {
  if (!clientIDFilter.value) return true
  return clientOptions.value.some(
    (c) => c.clientID === clientIDFilter.value && c.user === userFilter.value,
  )
})

const filteredProxies = computed(() => {
  let result = proxies.value

  // Filter by clientID and user if specified
  if (clientIDFilter.value) {
    result = result.filter(
      (p) => p.clientID === clientIDFilter.value && p.user === userFilter.value,
    )
  }

  // Filter by search text
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter((p) => p.name.toLowerCase().includes(search))
  }

  return result
})

const onClientFilterChange = (key: string) => {
  if (key) {
    const client = clientOptions.value.find((c) => c.key === key)
    if (client) {
      router.replace({
        query: { ...route.query, clientID: client.clientID, user: client.user },
      })
    }
  } else {
    const query = { ...route.query }
    delete query.clientID
    delete query.user
    router.replace({ query })
  }
}

const fetchClients = async () => {
  try {
    const json = await getClients()
    clients.value = json.map((data) => new Client(data))
  } catch {
    // Ignore errors when fetching clients
  }
}

// Server info cache
let serverInfo: {
  vhostHTTPPort: number
  vhostHTTPSPort: number
  tcpmuxHTTPConnectPort: number
  subdomainHost: string
} | null = null

const fetchServerInfo = async () => {
  if (serverInfo) return serverInfo
  const res = await getServerInfo()
  serverInfo = res
  return serverInfo
}

const fetchData = async () => {
  loading.value = true
  proxies.value = []

  try {
    const type = activeType.value
    const json = await getProxiesByType(type)

    if (type === 'tcp') {
      proxies.value = json.proxies.map((p: any) => new TCPProxy(p))
    } else if (type === 'udp') {
      proxies.value = json.proxies.map((p: any) => new UDPProxy(p))
    } else if (type === 'http') {
      const info = await fetchServerInfo()
      if (info && info.vhostHTTPPort) {
        proxies.value = json.proxies.map(
          (p: any) => new HTTPProxy(p, info.vhostHTTPPort, info.subdomainHost),
        )
      }
    } else if (type === 'https') {
      const info = await fetchServerInfo()
      if (info && info.vhostHTTPSPort) {
        proxies.value = json.proxies.map(
          (p: any) =>
            new HTTPSProxy(p, info.vhostHTTPSPort, info.subdomainHost),
        )
      }
    } else if (type === 'tcpmux') {
      const info = await fetchServerInfo()
      if (info && info.tcpmuxHTTPConnectPort) {
        proxies.value = json.proxies.map(
          (p: any) =>
            new TCPMuxProxy(p, info.tcpmuxHTTPConnectPort, info.subdomainHost),
        )
      }
    } else if (type === 'stcp') {
      proxies.value = json.proxies.map((p: any) => new STCPProxy(p))
    } else if (type === 'sudp') {
      proxies.value = json.proxies.map((p: any) => new SUDPProxy(p))
    }
  } catch (error: any) {
    ElMessage({
      showClose: true,
      message: 'Failed to fetch proxies: ' + error.message,
      type: 'error',
    })
  } finally {
    loading.value = false
  }
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

// Watch for type changes
watch(activeType, (newType) => {
  // Update route but preserve query params
  router.replace({ params: { type: newType }, query: route.query })
  fetchData()
})

// Watch for route query changes (client filter)
watch(
  () => [route.query.clientID, route.query.user],
  ([newClientID, newUser]) => {
    clientIDFilter.value = (newClientID as string) || ''
    userFilter.value = (newUser as string) || ''
  },
)

// Initial fetch
fetchData()
fetchClients()
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

.main-search :deep(.el-input__wrapper),
.client-filter :deep(.el-input__wrapper) {
  height: 32px;
  border-radius: 8px;
}

.client-filter {
  width: 240px;
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

@media (max-width: 768px) {
  .search-row {
    flex-direction: column;
  }

  .client-filter {
    width: 100%;
  }
}
</style>
