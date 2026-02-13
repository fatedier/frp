<template>
  <div class="overview-page">
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Total Proxies"
          :value="stats.total"
          type="proxies"
          subtitle="Configured proxies"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Running"
          :value="stats.running"
          type="running"
          subtitle="Active connections"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Error"
          :value="stats.error"
          type="error"
          subtitle="Failed proxies"
        />
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <StatCard
          label="Configure"
          value="Edit"
          type="config"
          subtitle="Manage settings"
          to="/configure"
        />
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :lg="16">
        <el-card class="proxy-list-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-left">
                <span class="card-title">Proxy Status</span>
                <el-tag size="small" type="info"
                  >{{ stats.total }} proxies</el-tag
                >
              </div>
              <div class="header-actions">
                <el-select
                  v-model="filterSource"
                  placeholder="Source"
                  clearable
                  class="filter-select"
                >
                  <el-option label="Config" value="config" />
                  <el-option label="Store" value="store" />
                </el-select>
                <el-select
                  v-model="filterType"
                  placeholder="Type"
                  clearable
                  class="filter-select"
                >
                  <el-option
                    v-for="type in availableTypes"
                    :key="type"
                    :label="type.toUpperCase()"
                    :value="type"
                  />
                </el-select>
                <el-input
                  v-model="searchText"
                  placeholder="Search..."
                  :prefix-icon="Search"
                  clearable
                  class="search-input"
                />
                <el-tooltip content="Refresh" placement="top">
                  <el-button :icon="Refresh" circle @click="fetchData" />
                </el-tooltip>
                <el-tooltip
                  v-if="storeEnabled"
                  content="Add new proxy"
                  placement="top"
                >
                  <el-button
                    type="primary"
                    :icon="Plus"
                    circle
                    @click="handleCreate"
                  />
                </el-tooltip>
              </div>
            </div>
          </template>

          <div v-loading="loading" class="proxy-list-content">
            <div v-if="filteredStatus.length > 0" class="proxy-list">
              <ProxyCard
                v-for="proxy in filteredStatus"
                :key="proxy.name"
                :proxy="proxy"
                @edit="handleEdit"
                @delete="handleDelete"
              />
            </div>
            <div v-else-if="!loading" class="empty-state">
              <div class="empty-content">
                <div class="empty-icon">
                  <svg
                    viewBox="0 0 64 64"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <rect
                      x="8"
                      y="16"
                      width="48"
                      height="32"
                      rx="4"
                      stroke="currentColor"
                      stroke-width="2"
                    />
                    <circle cx="20" cy="32" r="4" fill="currentColor" />
                    <circle cx="32" cy="32" r="4" fill="currentColor" />
                    <circle cx="44" cy="32" r="4" fill="currentColor" />
                  </svg>
                </div>
                <p class="empty-text">No proxies configured</p>
                <p class="empty-hint">
                  Add proxies in your configuration file or use Store to create
                  dynamic proxies
                </p>
                <el-button
                  v-if="storeEnabled"
                  type="primary"
                  :icon="Plus"
                  @click="handleCreate"
                >
                  Create First Proxy
                </el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="8">
        <el-card class="types-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Proxy Types</span>
              <el-tag size="small" type="info">Distribution</el-tag>
            </div>
          </template>
          <div class="proxy-types-grid">
            <div
              v-for="(count, type) in proxyTypeCounts"
              :key="type"
              class="proxy-type-item"
              :class="{ active: filterType === type }"
              v-show="count > 0"
              @click="toggleTypeFilter(String(type))"
            >
              <div class="proxy-type-name">
                {{ String(type).toUpperCase() }}
              </div>
              <div class="proxy-type-count">{{ count }}</div>
            </div>
            <div v-if="!hasActiveProxies" class="no-data">No proxy data</div>
          </div>
        </el-card>

        <el-card class="status-summary-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Status Summary</span>
            </div>
          </template>
          <div class="status-list">
            <div class="status-item">
              <div class="status-indicator running"></div>
              <span class="status-name">Running</span>
              <span class="status-count">{{ stats.running }}</span>
            </div>
            <div class="status-item">
              <div class="status-indicator waiting"></div>
              <span class="status-name">Waiting</span>
              <span class="status-count">{{ stats.waiting }}</span>
            </div>
            <div class="status-item">
              <div class="status-indicator error"></div>
              <span class="status-name">Error</span>
              <span class="status-count">{{ stats.error }}</span>
            </div>
          </div>
        </el-card>

        <!-- Store Status Card -->
        <el-card class="store-status-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Store</span>
              <el-tag
                size="small"
                :type="storeEnabled ? 'success' : 'info'"
                effect="plain"
              >
                {{ storeEnabled ? 'Enabled' : 'Disabled' }}
              </el-tag>
            </div>
          </template>
          <div class="store-info">
            <template v-if="storeEnabled">
              <div class="store-stat">
                <span class="store-stat-label">Store Proxies</span>
                <span class="store-stat-value">{{ storeProxies.length }}</span>
              </div>
              <div class="store-stat">
                <span class="store-stat-label">Store Visitors</span>
                <span class="store-stat-value">{{ storeVisitors.length }}</span>
              </div>
              <p class="store-hint">
                Proxies from Store are marked with a purple indicator
              </p>
            </template>
            <template v-else>
              <p class="store-disabled-text">
                Enable Store in your configuration to dynamically manage proxies
              </p>
            </template>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Store Visitors Section -->
    <el-row v-if="storeEnabled && storeVisitors.length > 0" :gutter="20">
      <el-col :span="24">
        <el-card class="visitors-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-left">
                <span class="card-title">Store Visitors</span>
                <el-tag size="small" type="info">{{ storeVisitors.length }} visitors</el-tag>
              </div>
              <el-tooltip content="Add new visitor" placement="top">
                <el-button
                  type="primary"
                  :icon="Plus"
                  circle
                  @click="handleCreateVisitor"
                />
              </el-tooltip>
            </div>
          </template>
          <div class="visitor-list">
            <div
              v-for="visitor in storeVisitors"
              :key="visitor.name"
              class="visitor-card"
            >
              <div class="visitor-card-header">
                <div class="visitor-info">
                  <span class="visitor-name">{{ visitor.name }}</span>
                  <el-tag size="small" type="info">{{ visitor.type.toUpperCase() }}</el-tag>
                </div>
                <div class="visitor-actions">
                  <el-button size="small" @click="handleEditVisitor(visitor)">
                    Edit
                  </el-button>
                  <el-button
                    size="small"
                    type="danger"
                    @click="handleDeleteVisitor(visitor.name)"
                  >
                    Delete
                  </el-button>
                </div>
              </div>
              <div class="visitor-card-body">
                <span v-if="visitor.config?.serverName">
                  Server: {{ visitor.config.serverName }}
                </span>
                <span v-if="visitor.config?.bindAddr || visitor.config?.bindPort">
                  Bind: {{ visitor.config.bindAddr || '127.0.0.1' }}:{{ visitor.config.bindPort }}
                </span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import {
  getStatus,
  listStoreProxies,
  deleteStoreProxy,
  listStoreVisitors,
  deleteStoreVisitor,
} from '../api/frpc'
import type {
  ProxyStatus,
  StoreProxyConfig,
  StoreVisitorConfig,
} from '../types/proxy'
import StatCard from '../components/StatCard.vue'
import ProxyCard from '../components/ProxyCard.vue'

const router = useRouter()

// State
const status = ref<ProxyStatus[]>([])
const storeProxies = ref<StoreProxyConfig[]>([])
const storeVisitors = ref<StoreVisitorConfig[]>([])
const storeEnabled = ref(false)
const loading = ref(false)
const searchText = ref('')
const filterSource = ref('')
const filterType = ref('')

// Computed
const stats = computed(() => {
  const total = status.value.length
  const running = status.value.filter((p) => p.status === 'running').length
  const error = status.value.filter((p) => p.status === 'error').length
  const waiting = total - running - error
  return { total, running, error, waiting }
})

const proxyTypeCounts = computed(() => {
  const counts: Record<string, number> = {}
  status.value.forEach((p) => {
    counts[p.type] = (counts[p.type] || 0) + 1
  })
  return counts
})

const hasActiveProxies = computed(() => {
  return status.value.length > 0
})

const availableTypes = computed(() => {
  const types = new Set<string>()
  status.value.forEach((p) => types.add(p.type))
  return Array.from(types).sort()
})

const filteredStatus = computed(() => {
  let result = status.value

  if (filterSource.value) {
    if (filterSource.value === 'store') {
      result = result.filter((p) => p.source === 'store')
    } else {
      result = result.filter((p) => !p.source || p.source !== 'store')
    }
  }

  if (filterType.value) {
    result = result.filter((p) => p.type === filterType.value)
  }

  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter(
      (p) =>
        p.name.toLowerCase().includes(search) ||
        p.type.toLowerCase().includes(search) ||
        p.local_addr.toLowerCase().includes(search) ||
        p.remote_addr.toLowerCase().includes(search),
    )
  }

  return result
})


// Methods
const toggleTypeFilter = (type: string) => {
  filterType.value = filterType.value === type ? '' : type
}

const fetchStatus = async () => {
  try {
    const json = await getStatus()
    const list: ProxyStatus[] = []
    for (const key in json) {
      for (const ps of json[key]) {
        list.push(ps)
      }
    }
    status.value = list
  } catch (err: any) {
    ElMessage.error('Failed to get status: ' + err.message)
  }
}

const fetchStoreProxies = async () => {
  try {
    const res = await listStoreProxies()
    storeProxies.value = res.proxies || []
    storeEnabled.value = true
  } catch (err: any) {
    if (err.status === 404) {
      storeEnabled.value = false
      storeProxies.value = []
    } else {
      console.error('Failed to fetch store proxies:', err)
    }
  }
}

const fetchStoreVisitors = async () => {
  try {
    const res = await listStoreVisitors()
    storeVisitors.value = res.visitors || []
  } catch (err: any) {
    if (err.status === 404) {
      storeVisitors.value = []
    } else {
      console.error('Failed to fetch store visitors:', err)
    }
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    await fetchStoreProxies()
    await fetchStoreVisitors()
    await fetchStatus()
  } finally {
    loading.value = false
  }
}

const handleCreate = () => {
  router.push('/proxies/create')
}

const handleEdit = (proxy: ProxyStatus) => {
  if (proxy.source !== 'store') return
  router.push('/proxies/' + encodeURIComponent(proxy.name) + '/edit')
}

const handleDelete = (proxy: ProxyStatus) => {
  if (proxy.source !== 'store') return

  ElMessageBox.confirm(
    `Are you sure you want to delete "${proxy.name}"? This action cannot be undone.`,
    'Delete Proxy',
    {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning',
      confirmButtonClass: 'el-button--danger',
    },
  ).then(async () => {
    try {
      await deleteStoreProxy(proxy.name)
      ElMessage.success('Proxy deleted')
      fetchData()
    } catch (err: any) {
      ElMessage.error('Delete failed: ' + err.message)
    }
  })
}

const handleCreateVisitor = () => {
  router.push('/visitors/create')
}

const handleEditVisitor = (visitor: StoreVisitorConfig) => {
  router.push('/visitors/' + encodeURIComponent(visitor.name) + '/edit')
}

const handleDeleteVisitor = async (name: string) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete visitor "${name}"? This action cannot be undone.`,
      'Delete Visitor',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      }
    )
    await deleteStoreVisitor(name)
    ElMessage.success('Visitor deleted')
    fetchData()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error('Delete failed: ' + (err.message || 'Unknown error'))
    }
  }
}

// Initial load
fetchData()
</script>

<style scoped>
.overview-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-row {
  margin-bottom: 0;
}

.stats-row .el-col {
  margin-bottom: 20px;
}

.content-row .el-col {
  margin-bottom: 20px;
}

.proxy-list-card,
.types-card,
.status-summary-card,
.store-status-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
}

html.dark .proxy-list-card,
html.dark .types-card,
html.dark .status-summary-card,
html.dark .store-status-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.status-summary-card,
.store-status-card {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

html.dark .card-title {
  color: #e5e7eb;
}

.filter-select {
  width: 100px;
}

.search-input {
  width: 180px;
}

.proxy-list-content {
  min-height: 200px;
}

.proxy-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Empty State */
.empty-state {
  padding: 48px 24px;
}

.empty-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.empty-icon {
  width: 80px;
  height: 80px;
  margin-bottom: 20px;
  color: #c0c4cc;
}

html.dark .empty-icon {
  color: #4b5563;
}

.empty-text {
  font-size: 16px;
  font-weight: 500;
  color: #606266;
  margin: 0 0 8px;
}

html.dark .empty-text {
  color: #9ca3af;
}

.empty-hint {
  font-size: 14px;
  color: #909399;
  margin: 0 0 20px;
  max-width: 320px;
}

/* Proxy Types Grid */
.proxy-types-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  gap: 12px;
  min-height: 80px;
  align-content: center;
}

.proxy-type-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 8px;
  background: #f8f9fa;
  border-radius: 8px;
  transition: all 0.2s;
  cursor: pointer;
}

.proxy-type-item:hover {
  background: #f0f2f5;
  transform: translateY(-2px);
}

.proxy-type-item.active {
  background: var(--el-color-primary-light-8);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-5);
}

.proxy-type-item.active .proxy-type-name {
  color: var(--el-color-primary);
}

.proxy-type-item.active .proxy-type-count {
  color: var(--el-color-primary);
}

html.dark .proxy-type-item {
  background: #1e1e2d;
}

html.dark .proxy-type-item:hover {
  background: #2a2a3c;
}

html.dark .proxy-type-item.active {
  background: var(--el-color-primary-dark-2);
  box-shadow: 0 0 0 2px var(--el-color-primary);
}

.proxy-type-name {
  font-size: 11px;
  color: #909399;
  font-weight: 600;
  margin-bottom: 4px;
  letter-spacing: 0.5px;
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
  height: 80px;
  color: #909399;
  font-size: 14px;
}

/* Status Summary */
.status-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  transition: all 0.2s;
}

.status-item:hover {
  background: #f0f2f5;
}

html.dark .status-item {
  background: #1e1e2d;
}

html.dark .status-item:hover {
  background: #2a2a3c;
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-indicator.running {
  background: var(--el-color-success);
  box-shadow: 0 0 0 3px var(--el-color-success-light-8);
}

.status-indicator.waiting {
  background: var(--el-color-warning);
  box-shadow: 0 0 0 3px var(--el-color-warning-light-8);
}

.status-indicator.error {
  background: var(--el-color-danger);
  box-shadow: 0 0 0 3px var(--el-color-danger-light-8);
}

.status-name {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-regular);
  font-weight: 500;
}

.status-count {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

/* Store Status Card */
.store-info {
  min-height: 60px;
}

.store-stat {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: linear-gradient(
    135deg,
    rgba(102, 126, 234, 0.08) 0%,
    rgba(118, 75, 162, 0.08) 100%
  );
  border-radius: 8px;
  margin-bottom: 12px;
}

html.dark .store-stat {
  background: linear-gradient(
    135deg,
    rgba(129, 140, 248, 0.12) 0%,
    rgba(167, 139, 250, 0.12) 100%
  );
}

.store-stat-label {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

html.dark .store-stat-label {
  color: #9ca3af;
}

.store-stat-value {
  font-size: 24px;
  font-weight: 600;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

html.dark .store-stat-value {
  background: linear-gradient(135deg, #818cf8 0%, #a78bfa 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.store-hint {
  font-size: 12px;
  color: #909399;
  margin: 0;
  line-height: 1.5;
}

.store-disabled-text {
  font-size: 13px;
  color: #909399;
  margin: 0;
  line-height: 1.6;
}

/* Visitors Card */
.visitors-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  margin-top: 20px;
}

html.dark .visitors-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.visitor-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.visitor-card {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  transition: all 0.2s;
}

.visitor-card:hover {
  background: #f0f2f5;
}

html.dark .visitor-card {
  background: #1e1e2d;
}

html.dark .visitor-card:hover {
  background: #2a2a3c;
}

.visitor-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.visitor-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.visitor-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

html.dark .visitor-name {
  color: #e5e7eb;
}

.visitor-actions {
  display: flex;
  gap: 8px;
}

.visitor-card-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: #606266;
}

html.dark .visitor-card-body {
  color: #9ca3af;
}

@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-left {
    justify-content: space-between;
  }

  .header-actions {
    justify-content: space-between;
  }

  .search-input {
    flex: 1;
    width: auto;
  }

  .proxy-types-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 992px) {
  .status-summary-card,
  .store-status-card {
    margin-top: 0;
  }
}
</style>
