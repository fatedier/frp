<template>
  <div class="visitors-page">
    <!-- Header -->
    <div class="page-header">
      <h2 class="page-title">Visitors</h2>
    </div>

    <!-- Tab bar -->
    <div class="tab-bar">
      <div class="tab-buttons">
        <button class="tab-btn active">Store</button>
      </div>
      <div class="tab-actions">
        <ActionButton variant="outline" size="small" @click="fetchData">
          <el-icon><Refresh /></el-icon>
        </ActionButton>
        <ActionButton v-if="visitorStore.storeEnabled" size="small" @click="handleCreate">
          + New Visitor
        </ActionButton>
      </div>
    </div>

    <div v-loading="visitorStore.loading">
      <div v-if="!visitorStore.storeEnabled" class="store-disabled">
        <p>Store is not enabled. Add the following to your frpc configuration:</p>
        <pre class="config-hint">[store]
path = "./frpc_store.json"</pre>
      </div>

      <template v-else>
        <div class="filter-bar">
          <el-input v-model="searchText" placeholder="Search..." clearable class="search-input">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <FilterDropdown v-model="typeFilter" label="Type" :options="typeOptions" :min-width="140" :is-mobile="isMobile" />
        </div>

        <div v-if="filteredVisitors.length > 0" class="visitor-list">
          <div v-for="v in filteredVisitors" :key="v.name" class="visitor-card" @click="goToDetail(v.name)">
            <div class="card-left">
              <div class="card-header">
                <span class="visitor-name">{{ v.name }}</span>
                <span class="type-tag">{{ v.type.toUpperCase() }}</span>
              </div>
              <div v-if="getServerName(v)" class="card-meta">{{ getServerName(v) }}</div>
            </div>
            <div class="card-right">
              <div @click.stop>
                <PopoverMenu :width="120" placement="bottom-end">
                  <template #trigger>
                    <ActionButton variant="outline" size="small">
                      <el-icon><MoreFilled /></el-icon>
                    </ActionButton>
                  </template>
                  <PopoverMenuItem @click="handleEdit(v)">
                    <el-icon><Edit /></el-icon>
                    Edit
                  </PopoverMenuItem>
                  <PopoverMenuItem danger @click="handleDelete(v.name)">
                    <el-icon><Delete /></el-icon>
                    Delete
                  </PopoverMenuItem>
                </PopoverMenu>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <p class="empty-text">No visitors found</p>
          <p class="empty-hint">Click "New Visitor" to create one.</p>
        </div>
      </template>
    </div>

    <ConfirmDialog v-model="deleteDialog.visible" title="Delete Visitor"
      :message="deleteDialog.message" confirm-text="Delete" danger
      :loading="deleteDialog.loading" :is-mobile="isMobile" @confirm="doDelete" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search, Refresh, MoreFilled, Edit, Delete } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import FilterDropdown from '@shared/components/FilterDropdown.vue'
import PopoverMenu from '@shared/components/PopoverMenu.vue'
import PopoverMenuItem from '@shared/components/PopoverMenuItem.vue'
import ConfirmDialog from '@shared/components/ConfirmDialog.vue'
import { useVisitorStore } from '../stores/visitor'
import { useResponsive } from '../composables/useResponsive'
import type { VisitorDefinition } from '../types'

const { isMobile } = useResponsive()
const router = useRouter()
const visitorStore = useVisitorStore()

const searchText = ref('')
const typeFilter = ref('')

const deleteDialog = reactive({
  visible: false,
  message: '',
  loading: false,
  name: '',
})

const typeOptions = computed(() => {
  return [
    { label: 'STCP', value: 'stcp' },
    { label: 'SUDP', value: 'sudp' },
    { label: 'XTCP', value: 'xtcp' },
  ]
})

const filteredVisitors = computed(() => {
  let list = visitorStore.storeVisitors

  if (typeFilter.value) {
    list = list.filter((v) => v.type === typeFilter.value)
  }

  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter((v) => v.name.toLowerCase().includes(q))
  }

  return list
})

const getServerName = (v: VisitorDefinition): string => {
  const block = (v as any)[v.type]
  return block?.serverName || ''
}

const fetchData = () => {
  visitorStore.fetchStoreVisitors()
}

const handleCreate = () => {
  router.push('/visitors/create')
}

const handleEdit = (v: VisitorDefinition) => {
  router.push('/visitors/' + encodeURIComponent(v.name) + '/edit')
}

const goToDetail = (name: string) => {
  router.push('/visitors/detail/' + encodeURIComponent(name))
}

const handleDelete = (name: string) => {
  deleteDialog.name = name
  deleteDialog.message = `Are you sure you want to delete visitor "${name}"? This action cannot be undone.`
  deleteDialog.visible = true
}

const doDelete = async () => {
  deleteDialog.loading = true
  try {
    await visitorStore.deleteVisitor(deleteDialog.name)
    ElMessage.success('Visitor deleted')
    deleteDialog.visible = false
    fetchData()
  } catch (err: any) {
    ElMessage.error('Delete failed: ' + (err.message || 'Unknown error'))
  } finally {
    deleteDialog.loading = false
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped lang="scss">
.visitors-page {
  height: 100%;
  overflow-y: auto;
  padding: $spacing-xl 40px;
  max-width: 960px;
  margin: 0 auto;
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

.tab-btn {
  background: none;
  border: none;
  padding: $spacing-sm $spacing-xl;
  font-size: $font-size-md;
  color: $color-text-muted;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all $transition-fast;

  &:hover {
    color: $color-text-primary;
  }

  &.active {
    color: $color-text-primary;
    border-bottom-color: $color-text-primary;
    font-weight: $font-weight-medium;
  }
}

.tab-actions {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  margin-bottom: $spacing-xl;

  :deep(.search-input) {
    flex: 1;
    min-width: 150px;
  }
}

.visitor-list {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}

.visitor-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: $color-bg-primary;
  border: 1px solid $color-border-lighter;
  border-radius: $radius-md;
  padding: 14px 20px;
  cursor: pointer;
  transition: all $transition-medium;

  &:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
    border-color: $color-border;
  }
}

.card-left {
  @include flex-column;
  gap: $spacing-sm;
  flex: 1;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.visitor-name {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
}

.type-tag {
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  padding: 2px 8px;
  border-radius: 4px;
  background: $color-bg-muted;
  color: $color-text-secondary;
}

.card-meta {
  font-size: $font-size-sm;
  color: $color-text-muted;
}

.card-right {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-shrink: 0;
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

@include mobile {
  .visitors-page {
    padding: $spacing-lg;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: $spacing-md;
  }

  .filter-bar {
    flex-wrap: wrap;

    :deep(.search-input) {
      flex: 1 1 100%;
    }
  }

  .visitor-card {
    flex-direction: column;
    align-items: stretch;
    gap: $spacing-sm;
  }

  .card-right {
    justify-content: flex-end;
  }
}
</style>
