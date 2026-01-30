<template>
  <div class="clients-page">
    <div class="page-header">
      <div class="header-top">
        <div class="title-section">
          <h1 class="page-title">Clients</h1>
          <p class="page-subtitle">Manage connected clients and their status</p>
        </div>
        <div class="status-tabs">
          <button
            v-for="tab in statusTabs"
            :key="tab.value"
            class="status-tab"
            :class="{ active: statusFilter === tab.value }"
            @click="statusFilter = tab.value"
          >
            <span class="status-dot" :class="tab.value"></span>
            <span class="tab-label">{{ tab.label }}</span>
            <span class="tab-count">{{ tab.count }}</span>
          </button>
        </div>
      </div>

      <div class="search-section">
        <el-input
          v-model="searchText"
          placeholder="Search clients..."
          :prefix-icon="Search"
          clearable
          class="search-input"
        />
      </div>
    </div>

    <div v-loading="loading" class="clients-content">
      <div v-if="filteredClients.length > 0" class="clients-list">
        <ClientCard
          v-for="client in filteredClients"
          :key="client.key"
          :client="client"
        />
      </div>
      <div v-else-if="!loading" class="empty-state">
        <el-empty description="No clients found" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { Client } from '../utils/client'
import ClientCard from '../components/ClientCard.vue'
import { getClients } from '../api/client'

const clients = ref<Client[]>([])
const loading = ref(false)
const searchText = ref('')
const statusFilter = ref<'all' | 'online' | 'offline'>('all')

let refreshTimer: number | null = null

const stats = computed(() => {
  const total = clients.value.length
  const online = clients.value.filter((c) => c.online).length
  const offline = total - online
  return { total, online, offline }
})

const statusTabs = computed(() => [
  { value: 'all' as const, label: 'All', count: stats.value.total },
  { value: 'online' as const, label: 'Online', count: stats.value.online },
  { value: 'offline' as const, label: 'Offline', count: stats.value.offline },
])

const filteredClients = computed(() => {
  let result = clients.value

  // Filter by status
  if (statusFilter.value === 'online') {
    result = result.filter((c) => c.online)
  } else if (statusFilter.value === 'offline') {
    result = result.filter((c) => !c.online)
  }

  // Filter by search text
  if (searchText.value) {
    result = result.filter((c) => c.matchesFilter(searchText.value))
  }

  // Sort: online first, then by display name
  result.sort((a, b) => {
    if (a.online !== b.online) {
      return a.online ? -1 : 1
    }
    return a.displayName.localeCompare(b.displayName)
  })

  return result
})

const fetchData = async () => {
  loading.value = true
  try {
    const json = await getClients()
    clients.value = json.map((data) => new Client(data))
  } catch (error: any) {
    ElMessage({
      showClose: true,
      message: 'Failed to fetch clients: ' + error.message,
      type: 'error',
    })
  } finally {
    loading.value = false
  }
}

const startAutoRefresh = () => {
  refreshTimer = window.setInterval(() => {
    fetchData()
  }, 5000)
}

const stopAutoRefresh = () => {
  if (refreshTimer !== null) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  fetchData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.clients-page {
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
  align-items: flex-end;
  gap: 20px;
  flex-wrap: wrap;
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

.status-tabs {
  display: flex;
  gap: 12px;
}

.status-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid var(--el-border-color);
  border-radius: 20px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.status-tab:hover {
  border-color: var(--el-border-color-darker);
  background: var(--el-fill-color-light);
}

.status-tab.active {
  background: var(--el-fill-color-dark);
  border-color: var(--el-text-color-primary);
  color: var(--el-text-color-primary);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--el-text-color-secondary);
}

.status-dot.online {
  background-color: var(--el-color-success);
}

.status-dot.offline {
  background-color: var(--el-text-color-placeholder);
}

.status-dot.all {
  background-color: var(--el-text-color-regular);
}

.tab-count {
  font-weight: 500;
  opacity: 0.8;
}

.search-section {
  width: 100%;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  padding: 8px 16px;
  border: 1px solid var(--el-border-color);
  transition: all 0.2s;
  height: 48px;
  font-size: 15px;
}

.search-input :deep(.el-input__wrapper:hover) {
  border-color: var(--el-border-color-darker);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.06);
}

.search-input :deep(.el-input__wrapper.is-focus) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary);
}

.clients-content {
  min-height: 200px;
}

.clients-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.empty-state {
  padding: 60px 0;
}

/* Dark mode adjustments */
html.dark .status-tab {
  background: var(--el-bg-color-overlay);
}

html.dark .status-tab.active {
  background: var(--el-fill-color);
}

@media (max-width: 640px) {
  .header-top {
    flex-direction: column;
    align-items: flex-start;
  }

  .status-tabs {
    width: 100%;
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .status-tab {
    flex-shrink: 0;
  }
}
</style>
