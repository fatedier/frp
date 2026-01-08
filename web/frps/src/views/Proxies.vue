<template>
  <div class="proxies-page">
    <!-- Main Content -->
    <el-card class="main-card" shadow="never">
      <div class="toolbar-header">
        <el-tabs v-model="activeType" class="proxy-tabs">
          <el-tab-pane
            v-for="t in proxyTypes"
            :key="t.value"
            :label="t.label"
            :name="t.value"
          />
        </el-tabs>

        <div class="toolbar-actions">
          <el-input
            v-model="searchText"
            placeholder="Search by name..."
            :prefix-icon="Search"
            clearable
            class="search-input"
          />
          <el-tooltip content="Refresh" placement="top">
            <el-button :icon="Refresh" circle @click="fetchData" />
          </el-tooltip>
          <el-popconfirm
            title="Are you sure to clear all data of offline proxies?"
            @confirm="clearOfflineProxies"
          >
            <template #reference>
              <el-button type="danger" plain :icon="Delete"
                >Clear Offline</el-button
              >
            </template>
          </el-popconfirm>
        </div>
      </div>

      <el-table
        v-loading="loading"
        :data="filteredProxies"
        :default-sort="{ prop: 'name', order: 'ascending' }"
        style="width: 100%"
      >
        <el-table-column type="expand">
          <template #default="props">
            <div class="expand-wrapper">
              <ProxyViewExpand :row="props.row" :proxyType="activeType" />
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="Name"
          prop="name"
          sortable
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column label="Port" prop="port" sortable width="100" />
        <el-table-column
          label="Conns"
          prop="conns"
          sortable
          width="100"
          align="center"
        />
        <el-table-column label="Traffic" width="220">
          <template #default="scope">
            <div class="traffic-cell">
              <span class="traffic-item up" title="Traffic Out">
                <el-icon><Top /></el-icon>
                {{ formatFileSize(scope.row.trafficOut) }}
              </span>
              <span class="traffic-item down" title="Traffic In">
                <el-icon><Bottom /></el-icon>
                {{ formatFileSize(scope.row.trafficIn) }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="Version"
          prop="clientVersion"
          sortable
          width="140"
          show-overflow-tooltip
        />
        <el-table-column
          label="Status"
          prop="status"
          sortable
          width="120"
          align="center"
        >
          <template #default="scope">
            <el-tag
              :type="scope.row.status === 'online' ? 'success' : 'danger'"
              effect="light"
              round
            >
              {{ scope.row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="Action"
          width="120"
          align="center"
          fixed="right"
        >
          <template #default="scope">
            <el-button
              type="primary"
              link
              :icon="DataAnalysis"
              @click="showTraffic(scope.row.name)"
            >
              Traffic
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      destroy-on-close
      :title="`Traffic Statistics - ${dialogVisibleName}`"
      width="700px"
      align-center
      class="traffic-dialog"
    >
      <Traffic :proxyName="dialogVisibleName" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { formatFileSize } from '../utils/format'
import { ElMessage } from 'element-plus'
import {
  Search,
  Refresh,
  Delete,
  Top,
  Bottom,
  DataAnalysis,
} from '@element-plus/icons-vue'
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
import ProxyViewExpand from '../components/ProxyViewExpand.vue'
import Traffic from '../components/Traffic.vue'
import { getProxiesByType, clearOfflineProxies as apiClearOfflineProxies } from '../api/proxy'
import { getServerInfo } from '../api/server'

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
const loading = ref(false)
const searchText = ref('')
const dialogVisible = ref(false)
const dialogVisibleName = ref('')

const filteredProxies = computed(() => {
  if (!searchText.value) {
    return proxies.value
  }
  const search = searchText.value.toLowerCase()
  return proxies.value.filter((p) => p.name.toLowerCase().includes(search))
})

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
    console.error('Failed to fetch proxies:', error)
    ElMessage({
      showClose: true,
      message: 'Failed to fetch proxies: ' + error.message,
      type: 'error',
    })
  } finally {
    loading.value = false
  }
}

const showTraffic = (name: string) => {
  dialogVisibleName.value = name
  dialogVisible.value = true
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
  router.replace({ params: { type: newType } })
  fetchData()
})

// Initial fetch
fetchData()
</script>

<style scoped>
.proxies-page {
  padding: 24px;
  max-width: 1600px;
  margin: 0 auto;
}

/* Main Content */
.main-card {
  border-radius: 12px;
  border: none;
}

.toolbar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding-bottom: 16px;
}

.proxy-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.proxy-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 0;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-input {
  width: 240px;
}

/* Table Styling */
.traffic-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.traffic-item.up {
  color: #67c23a;
}
.traffic-item.down {
  color: #409eff;
}

.expand-wrapper {
  padding: 16px 24px;
  background-color: transparent;
}

/* Responsive */
@media (max-width: 768px) {
  .toolbar-header {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-actions {
    justify-content: space-between;
  }

  .search-input {
    flex: 1;
  }
}
</style>
