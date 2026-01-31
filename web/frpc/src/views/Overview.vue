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
              </div>
            </div>
          </template>

          <div v-loading="loading" class="proxy-list-content">
            <div v-if="filteredStatus.length > 0" class="proxy-list">
              <ProxyCard
                v-for="proxy in filteredStatus"
                :key="proxy.name"
                :proxy="proxy"
              />
            </div>
            <div v-else-if="!loading" class="empty-state">
              <el-empty description="No proxies found" />
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
              v-show="count > 0"
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
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import { getStatus } from '../api/frpc'
import type { ProxyStatus } from '../types/proxy'
import StatCard from '../components/StatCard.vue'
import ProxyCard from '../components/ProxyCard.vue'

const status = ref<ProxyStatus[]>([])
const loading = ref(false)
const searchText = ref('')

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

const filteredStatus = computed(() => {
  if (!searchText.value) {
    return status.value
  }
  const search = searchText.value.toLowerCase()
  return status.value.filter(
    (p) =>
      p.name.toLowerCase().includes(search) ||
      p.type.toLowerCase().includes(search) ||
      p.local_addr.toLowerCase().includes(search) ||
      p.remote_addr.toLowerCase().includes(search),
  )
})

const fetchData = async () => {
  loading.value = true
  try {
    const json = await getStatus()
    status.value = []
    for (const key in json) {
      for (const ps of json[key]) {
        status.value.push(ps)
      }
    }
  } catch (err: any) {
    ElMessage({
      showClose: true,
      message: 'Get status info from frpc failed! ' + err.message,
      type: 'warning',
    })
  } finally {
    loading.value = false
  }
}

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
.status-summary-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
}

html.dark .proxy-list-card,
html.dark .types-card,
html.dark .status-summary-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.status-summary-card {
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
  gap: 12px;
}

.card-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

html.dark .card-title {
  color: #e5e7eb;
}

.search-input {
  width: 200px;
}

.proxy-list-content {
  min-height: 200px;
}

.proxy-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-state {
  padding: 40px 0;
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
}

.proxy-type-item:hover {
  background: #f0f2f5;
  transform: translateY(-2px);
}

html.dark .proxy-type-item {
  background: #1e1e2d;
}

html.dark .proxy-type-item:hover {
  background: #2a2a3c;
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
  .status-summary-card {
    margin-top: 0;
  }
}
</style>
