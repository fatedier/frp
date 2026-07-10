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
            <span v-if="tab.count !== null" class="tab-count">{{
              tab.count
            }}</span>
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
      <div v-if="clients.length > 0" class="clients-list">
        <ClientCard
          v-for="client in clients"
          :key="client.key"
          :client="client"
        />
      </div>
      <div v-else-if="!loading" class="empty-state">
        <el-empty description="No clients found" />
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

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElPagination } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { Client } from '../utils/client'
import ClientCard from '../components/ClientCard.vue'
import { getClientsV2 } from '../api/client'

const clients = ref<Client[]>([])
const loading = ref(false)
const searchText = ref('')
const statusFilter = ref<'all' | 'online' | 'offline'>('all')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

let refreshTimer: number | null = null
let searchDebounceTimer: number | null = null
let requestSeq = 0

const statusTabs = computed(() => [
  {
    value: 'all' as const,
    label: 'All',
    count: statusFilter.value === 'all' ? total.value : null,
  },
  {
    value: 'online' as const,
    label: 'Online',
    count: statusFilter.value === 'online' ? total.value : null,
  },
  {
    value: 'offline' as const,
    label: 'Offline',
    count: statusFilter.value === 'offline' ? total.value : null,
  },
])

const fetchData = async (silent = false) => {
  const seq = ++requestSeq
  if (!silent) loading.value = true
  try {
    const data = await getClientsV2({
      page: page.value,
      pageSize: pageSize.value,
      status: statusFilter.value,
      q: searchText.value.trim(),
    })
    if (seq !== requestSeq) return

    const maxPage = Math.max(1, Math.ceil(data.total / data.pageSize))
    if (data.items.length === 0 && data.total > 0 && data.page > maxPage) {
      page.value = maxPage
      await fetchData(silent)
      return
    }

    clients.value = data.items.map((item) => new Client(item))
    total.value = data.total
    page.value = data.page
    pageSize.value = data.pageSize
  } catch (error: any) {
    if (seq !== requestSeq) return
    ElMessage({
      showClose: true,
      message: 'Failed to fetch clients: ' + error.message,
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

const onPageChange = (value: number) => {
  clearSearchDebounce()
  page.value = value
  fetchData()
}

const onPageSizeChange = (value: number) => {
  pageSize.value = value
  resetPageAndFetch()
}

const startAutoRefresh = () => {
  refreshTimer = window.setInterval(() => {
    fetchData(true)
  }, 5000)
}

const stopAutoRefresh = () => {
  if (refreshTimer !== null) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
}

watch(statusFilter, () => {
  resetPageAndFetch()
})

watch(searchText, () => {
  clearSearchDebounce()
  page.value = 1
  searchDebounceTimer = window.setTimeout(() => {
    searchDebounceTimer = null
    fetchData()
  }, 300)
})

onMounted(() => {
  fetchData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
  clearSearchDebounce()
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

.pagination-section {
  display: flex;
  justify-content: flex-end;
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

  .pagination-section {
    justify-content: center;
  }
}
</style>
