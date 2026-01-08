<template>
  <div class="overview-page">
    <el-card class="main-card" shadow="never">
      <div class="toolbar-header">
        <h2 class="card-title">Proxy Status</h2>
        <div class="toolbar-actions">
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

      <el-table
        v-loading="loading"
        :data="filteredStatus"
        :default-sort="{ prop: 'name', order: 'ascending' }"
        stripe
        style="width: 100%"
        class="proxy-table"
      >
        <el-table-column
          prop="name"
          label="Name"
          sortable
          min-width="120"
        ></el-table-column>
        <el-table-column
          prop="type"
          label="Type"
          width="100"
          sortable
        >
          <template #default="scope">
            <span class="type-text">{{ scope.row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column
          prop="local_addr"
          label="Local Address"
          min-width="150"
          sortable
          show-overflow-tooltip
        ></el-table-column>
        <el-table-column
          prop="plugin"
          label="Plugin"
          width="120"
          sortable
          show-overflow-tooltip
        ></el-table-column>
        <el-table-column
          prop="remote_addr"
          label="Remote Address"
          min-width="150"
          sortable
          show-overflow-tooltip
        ></el-table-column>
        <el-table-column
          prop="status"
          label="Status"
          width="120"
          sortable
          align="center"
        >
          <template #default="scope">
            <el-tag
              :type="getStatusColor(scope.row.status)"
              effect="light"
              round
            >
              {{ scope.row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="err" label="Info" min-width="150" show-overflow-tooltip>
             <template #default="scope">
                <span v-if="scope.row.err" class="error-text">{{ scope.row.err }}</span>
                <span v-else>-</span>
             </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import { getStatus } from '../api/frpc'
import type { ProxyStatus } from '../types/proxy'

const status = ref<ProxyStatus[]>([])
const loading = ref(false)
const searchText = ref('')

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
      p.remote_addr.toLowerCase().includes(search)
  )
})

const getStatusColor = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'error':
      return 'danger'
    default:
      return 'warning'
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const json = await getStatus()
    status.value = []
    for (const key in json) {
      // json[key] is generic array, we assume it matches ProxyStatus
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
  /* No special padding needed if App.vue handles content padding */
}

.main-card {
  border-radius: 12px;
  border: none;
}

.card-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
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

.toolbar-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-input {
  width: 240px;
}

.error-text {
    color: var(--el-color-danger);
}

.type-text {
  display: inline-block;
  padding: 2px 8px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  color: var(--el-text-color-regular);
}

@media (max-width: 768px) {
  .toolbar-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-input {
    width: 100%;
  }
}
</style>