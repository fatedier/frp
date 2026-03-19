<template>
  <div class="proxy-detail-page">
    <!-- Fixed Header -->
    <div class="detail-top">
      <nav class="breadcrumb">
        <router-link :to="isStore ? '/proxies?tab=store' : '/proxies'" class="breadcrumb-link">Proxies</router-link>
        <span class="breadcrumb-sep">&rsaquo;</span>
        <span class="breadcrumb-current">{{ proxyName }}</span>
      </nav>

      <template v-if="proxy">
        <div class="detail-header">
          <div>
            <div class="header-title-row">
              <h2 class="detail-title">{{ proxy.name }}</h2>
              <span class="status-pill" :class="statusClass">
                <span class="status-dot"></span>
                {{ proxy.status }}
              </span>
            </div>
            <p class="header-subtitle">
              Source: {{ displaySource }} &middot; Type:
              {{ proxy.type.toUpperCase() }}
            </p>
          </div>
          <div v-if="isStore" class="header-actions">
            <ActionButton variant="outline" size="small" @click="handleEdit">
              Edit
            </ActionButton>
          </div>
        </div>
      </template>
    </div>

    <!-- Scrollable Content -->
    <div v-if="notFound" class="not-found">
      <p class="empty-text">Proxy not found</p>
      <p class="empty-hint">The proxy "{{ proxyName }}" does not exist.</p>
      <ActionButton variant="outline" @click="router.push('/proxies')">
        Back to Proxies
      </ActionButton>
    </div>

    <div v-else-if="proxy" v-loading="loading" class="detail-content">
      <!-- Error Banner -->
      <div v-if="proxy.err" class="error-banner">
        <el-icon class="error-icon"><Warning /></el-icon>
        <div>
          <div class="error-title">Connection Error</div>
          <div class="error-message">{{ proxy.err }}</div>
        </div>
      </div>

      <!-- Config Sections -->
      <ProxyFormLayout
        v-if="formData"
        :model-value="formData"
        readonly
      />
    </div>

    <div v-else v-loading="loading" class="loading-area"></div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Warning } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import ProxyFormLayout from '../components/proxy-form/ProxyFormLayout.vue'
import { getProxyConfig, getStoreProxy } from '../api/frpc'
import { useProxyStore } from '../stores/proxy'
import { storeProxyToForm } from '../types'
import type { ProxyStatus, ProxyDefinition, ProxyFormData } from '../types'

const route = useRoute()
const router = useRouter()
const proxyStore = useProxyStore()

const proxyName = route.params.name as string
const proxy = ref<ProxyStatus | null>(null)
const proxyConfig = ref<ProxyDefinition | null>(null)
const loading = ref(true)
const notFound = ref(false)
const isStore = ref(false)

onMounted(async () => {
  try {
    // Try status API first
    await proxyStore.fetchStatus()
    const found = proxyStore.proxies.find((p) => p.name === proxyName)

    // Try config API (works for any source)
    let configDef: ProxyDefinition | null = null
    try {
      configDef = await getProxyConfig(proxyName)
      proxyConfig.value = configDef
    } catch {
      // Config not available
    }

    // Check if proxy is from the store (for Edit/Delete buttons)
    try {
      await getStoreProxy(proxyName)
      isStore.value = true
    } catch {
      // Not a store proxy
    }

    if (found) {
      proxy.value = found
    } else if (configDef) {
      // Proxy not in status (e.g. disabled), build from config definition
      const block = (configDef as any)[configDef.type]
      const localIP = block?.localIP || '127.0.0.1'
      const localPort = block?.localPort
      const enabled = block?.enabled !== false
      proxy.value = {
        name: configDef.name,
        type: configDef.type,
        status: enabled ? 'waiting' : 'disabled',
        err: '',
        local_addr: localPort != null ? `${localIP}:${localPort}` : '',
        remote_addr: block?.remotePort != null ? `:${block.remotePort}` : '',
        plugin: block?.plugin?.type || '',
      }
    } else {
      notFound.value = true
    }
  } catch (err: any) {
    ElMessage.error('Failed to load proxy: ' + err.message)
  } finally {
    loading.value = false
  }
})

const displaySource = computed(() =>
  isStore.value ? 'store' : 'config',
)

const statusClass = computed(() => {
  const s = proxy.value?.status
  if (s === 'running') return 'running'
  if (s === 'error') return 'error'
  if (s === 'disabled') return 'disabled'
  return 'waiting'
})

const formData = computed((): ProxyFormData | null => {
  if (!proxyConfig.value) return null
  return storeProxyToForm(proxyConfig.value)
})

const handleEdit = () => {
  router.push('/proxies/' + encodeURIComponent(proxyName) + '/edit')
}

</script>

<style scoped lang="scss">
.proxy-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 960px;
  margin: 0 auto;
}

.detail-top {
  flex-shrink: 0;
  padding: $spacing-xl 24px 0;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 24px 160px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  font-size: $font-size-md;
  margin-bottom: $spacing-lg;
}

.breadcrumb-link {
  color: $color-text-secondary;
  text-decoration: none;

  &:hover {
    color: $color-text-primary;
  }
}

.breadcrumb-sep {
  color: $color-text-light;
}

.breadcrumb-current {
  color: $color-text-primary;
  font-weight: $font-weight-medium;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: $spacing-xl;
}

.header-title-row {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-sm;
}

.detail-title {
  margin: 0;
  font-size: 22px;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
}

.header-subtitle {
  font-size: $font-size-sm;
  color: $color-text-muted;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: $spacing-sm;
}

.error-banner {
  display: flex;
  align-items: flex-start;
  gap: $spacing-sm;
  padding: 12px 16px;
  background: var(--color-danger-light);
  border: 1px solid rgba(245, 108, 108, 0.2);
  border-radius: $radius-md;
  margin-bottom: $spacing-xl;

  .error-icon {
    color: $color-danger;
    font-size: 18px;
    margin-top: 2px;
  }

  .error-title {
    font-size: $font-size-md;
    font-weight: $font-weight-medium;
    color: $color-danger;
    margin-bottom: 2px;
  }

  .error-message {
    font-size: $font-size-sm;
    color: $color-text-muted;
  }
}

.not-found,
.loading-area {
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
  margin: 0 0 $spacing-lg;
}

@include mobile {
  .detail-top {
    padding: $spacing-xl $spacing-lg 0;
  }

  .detail-content {
    padding: 0 $spacing-lg $spacing-xl;
  }

  .detail-header {
    flex-direction: column;
    gap: $spacing-md;
  }
}
</style>
