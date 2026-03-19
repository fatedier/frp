<template>
  <div class="proxies-page">
    <!-- Fixed top area -->
    <div class="page-top">
      <!-- Header -->
      <div class="page-header">
        <h2 class="page-title">Proxies</h2>
      </div>

      <!-- Tabs -->
      <div class="tab-bar">
        <div class="tab-buttons">
          <button class="tab-btn" :class="{ active: activeTab === 'status' }" @click="switchTab('status')">Status</button>
          <button class="tab-btn" :class="{ active: activeTab === 'store' }" @click="switchTab('store')">Store</button>
        </div>
        <div class="tab-actions">
          <ActionButton variant="outline" size="small" @click="refreshData">
            <el-icon><Refresh /></el-icon>
          </ActionButton>
          <ActionButton v-if="activeTab === 'store' && proxyStore.storeEnabled" size="small" @click="handleCreate">
            + New Proxy
          </ActionButton>
        </div>
      </div>

      <!-- Status Tab Filters -->
      <template v-if="activeTab === 'status'">
        <StatusPills v-if="!isMobile" :items="proxyStore.proxies" v-model="statusFilter" />
        <div class="filter-bar">
          <el-input v-model="searchText" placeholder="Search..." clearable class="search-input">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <FilterDropdown v-model="sourceFilter" label="Source" :options="sourceOptions" :min-width="140" :is-mobile="isMobile" />
          <FilterDropdown v-model="typeFilter" label="Type" :options="typeOptions" :min-width="140" :is-mobile="isMobile" />
        </div>
      </template>

      <!-- Store Tab Filters -->
      <template v-if="activeTab === 'store' && proxyStore.storeEnabled">
        <div class="filter-bar">
          <el-input v-model="storeSearch" placeholder="Search..." clearable class="search-input">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <FilterDropdown v-model="storeTypeFilter" label="Type" :options="storeTypeOptions" :min-width="140" :is-mobile="isMobile" />
        </div>
      </template>
    </div>

    <!-- Scrollable list area -->
    <div class="page-content">
      <!-- Status Tab List -->
      <div v-if="activeTab === 'status'" v-loading="proxyStore.loading">
        <div v-if="filteredStatus.length > 0" class="proxy-list">
          <ProxyCard
            v-for="p in filteredStatus"
            :key="p.name"
            :proxy="p"
            showSource
            @click="goToDetail(p.name)"
          />
        </div>
        <div v-else-if="!proxyStore.loading" class="empty-state">
          <p class="empty-text">No proxies found</p>
          <p class="empty-hint">Proxies will appear here once configured and connected.</p>
        </div>
      </div>

      <!-- Store Tab List -->
      <div v-if="activeTab === 'store'" v-loading="proxyStore.storeLoading">
        <div v-if="!proxyStore.storeEnabled" class="store-disabled">
          <p>Store is not enabled. Add the following to your frpc configuration:</p>
          <pre class="config-hint">[store]
path = "./frpc_store.json"</pre>
        </div>
        <template v-else>
          <div v-if="filteredStoreProxies.length > 0" class="proxy-list">
            <ProxyCard
              v-for="p in filteredStoreProxies"
              :key="p.name"
              :proxy="proxyStore.storeProxyWithStatus(p)"
              showActions
              @click="goToDetail(p.name)"
              @edit="handleEdit"
              @toggle="handleToggleProxy"
              @delete="handleDeleteProxy(p.name)"
            />
          </div>
          <div v-else class="empty-state">
            <p class="empty-text">No store proxies</p>
            <p class="empty-hint">Click "New Proxy" to create one.</p>
          </div>
        </template>
      </div>
    </div>

    <ConfirmDialog
      v-model="deleteDialog.visible"
      title="Delete Proxy"
      :message="deleteDialog.message"
      confirm-text="Delete"
      danger
      :loading="deleteDialog.loading"
      :is-mobile="isMobile"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import StatusPills from '../components/StatusPills.vue'
import FilterDropdown from '@shared/components/FilterDropdown.vue'
import ProxyCard from '../components/ProxyCard.vue'
import ConfirmDialog from '@shared/components/ConfirmDialog.vue'
import { useProxyStore } from '../stores/proxy'
import { useResponsive } from '../composables/useResponsive'
import type { ProxyStatus } from '../types'

const { isMobile } = useResponsive()

const route = useRoute()
const router = useRouter()
const proxyStore = useProxyStore()

// Tab
const activeTab = computed(() => {
  const tab = route.query.tab as string
  return tab === 'store' ? 'store' : 'status'
})

const switchTab = (tab: string) => {
  router.replace({ query: tab === 'status' ? {} : { tab } })
}

// Filters (local UI state)
const statusFilter = ref('')
const typeFilter = ref('')
const sourceFilter = ref('')
const searchText = ref('')
const storeSearch = ref('')
const storeTypeFilter = ref('')

// Delete dialog
const deleteDialog = reactive({
  visible: false,
  title: 'Delete Proxy',
  message: '',
  loading: false,
  name: '',
})

// Source handling
const displaySource = (proxy: ProxyStatus): string => {
  return proxy.source === 'store' ? 'store' : 'config'
}

// Filter options
const sourceOptions = computed(() => {
  const sources = new Set<string>()
  sources.add('config')
  sources.add('store')
  proxyStore.proxies.forEach((p) => {
    sources.add(displaySource(p))
  })
  return Array.from(sources)
    .sort()
    .map((s) => ({ label: s, value: s }))
})

const PROXY_TYPE_ORDER = ['tcp', 'udp', 'http', 'https', 'tcpmux', 'stcp', 'sudp', 'xtcp']

const sortByTypeOrder = (types: string[]) => {
  return types.sort((a, b) => {
    const ia = PROXY_TYPE_ORDER.indexOf(a)
    const ib = PROXY_TYPE_ORDER.indexOf(b)
    return (ia === -1 ? 999 : ia) - (ib === -1 ? 999 : ib)
  })
}

const typeOptions = computed(() => {
  const types = new Set<string>()
  proxyStore.proxies.forEach((p) => types.add(p.type))
  return sortByTypeOrder(Array.from(types))
    .map((t) => ({ label: t.toUpperCase(), value: t }))
})

const storeTypeOptions = computed(() => {
  const types = new Set<string>()
  proxyStore.storeProxies.forEach((p) => types.add(p.type))
  return sortByTypeOrder(Array.from(types))
    .map((t) => ({ label: t.toUpperCase(), value: t }))
})

// Filtered computeds — Status tab uses proxyStore.proxies (runtime only)
const filteredStatus = computed(() => {
  let result = proxyStore.proxies as ProxyStatus[]

  if (statusFilter.value) {
    result = result.filter((p) => p.status === statusFilter.value)
  }

  if (typeFilter.value) {
    result = result.filter((p) => p.type === typeFilter.value)
  }

  if (sourceFilter.value) {
    result = result.filter((p) => displaySource(p) === sourceFilter.value)
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

const filteredStoreProxies = computed(() => {
  let list = proxyStore.storeProxies

  if (storeTypeFilter.value) {
    list = list.filter((p) => p.type === storeTypeFilter.value)
  }

  if (storeSearch.value) {
    const q = storeSearch.value.toLowerCase()
    list = list.filter((p) => p.name.toLowerCase().includes(q))
  }

  return list
})

// Data fetching
const refreshData = () => {
  proxyStore.fetchStatus().catch((err: any) => {
    ElMessage.error('Failed to get status: ' + err.message)
  })
  proxyStore.fetchStoreProxies()
}

// Navigation
const goToDetail = (name: string) => {
  router.push('/proxies/detail/' + encodeURIComponent(name))
}

const handleCreate = () => {
  router.push('/proxies/create')
}

const handleEdit = (proxy: ProxyStatus) => {
  router.push('/proxies/' + encodeURIComponent(proxy.name) + '/edit')
}

const handleToggleProxy = async (proxy: ProxyStatus, enabled: boolean) => {
  try {
    await proxyStore.toggleProxy(proxy.name, enabled)
    ElMessage.success(enabled ? 'Proxy enabled' : 'Proxy disabled')
  } catch (err: any) {
    ElMessage.error('Operation failed: ' + (err.message || 'Unknown error'))
  }
}

const handleDeleteProxy = (name: string) => {
  deleteDialog.name = name
  deleteDialog.message = `Are you sure you want to delete "${name}"? This action cannot be undone.`
  deleteDialog.visible = true
}

const doDelete = async () => {
  deleteDialog.loading = true
  try {
    await proxyStore.deleteProxy(deleteDialog.name)
    ElMessage.success('Proxy deleted')
    deleteDialog.visible = false
    proxyStore.fetchStatus()
  } catch (err: any) {
    ElMessage.error('Delete failed: ' + (err.message || 'Unknown error'))
  } finally {
    deleteDialog.loading = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped lang="scss">
.proxies-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 960px;
  margin: 0 auto;
}

.page-top {
  flex-shrink: 0;
  padding: $spacing-xl 40px 0;
}

.page-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 40px $spacing-xl;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-xl;
}

.tab-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid $color-border-lighter;
  margin-bottom: $spacing-xl;
}

.tab-buttons {
  display: flex;
}

.tab-actions {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.tab-btn {
  background: none;
  border: none;
  padding: $spacing-sm $spacing-xl;
  font-size: $font-size-md;
  color: $color-text-muted;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all $transition-fast;

  &:hover { color: $color-text-primary; }
  &.active {
    color: $color-text-primary;
    border-bottom-color: $color-text-primary;
    font-weight: $font-weight-medium;
  }
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin-top: $spacing-lg;
  padding-bottom: $spacing-lg;

  :deep(.search-input) {
    flex: 1;
    min-width: 150px;
  }
}

.proxy-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.empty-state {
  text-align: center;
  padding: 60px $spacing-xl;
}

.empty-text {
  font-size: $font-size-lg;
  font-weight: $font-weight-medium;
  color: $color-text-secondary;
  margin: 0 0 $spacing-xs;
}

.empty-hint {
  font-size: $font-size-sm;
  color: $color-text-muted;
  margin: 0;
}

.store-disabled {
  padding: 32px;
  text-align: center;
  color: $color-text-muted;
}

.config-hint {
  display: inline-block;
  text-align: left;
  background: $color-bg-hover;
  padding: 12px 20px;
  border-radius: $radius-sm;
  font-size: $font-size-sm;
  margin-top: $spacing-md;
}

@include mobile {
  .page-top {
    padding: $spacing-lg $spacing-lg 0;
  }

  .page-content {
    padding: 0 $spacing-lg $spacing-lg;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: $spacing-md;
  }

  .header-actions {
    justify-content: flex-end;
  }

  .filter-bar {
    :deep(.search-input) {
      flex: 1;
      min-width: 0;
    }
  }
}
</style>
