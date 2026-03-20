<template>
  <div class="proxy-edit-page">
    <!-- Header with breadcrumb and actions -->
    <div class="edit-header">
      <nav class="breadcrumb">
        <router-link to="/proxies?tab=store" class="breadcrumb-item">Proxies</router-link>
        <span class="breadcrumb-separator">&rsaquo;</span>
        <span class="breadcrumb-current">{{ isEditing ? 'Edit Proxy' : 'New Proxy' }}</span>
      </nav>
      <div class="header-actions">
        <ActionButton variant="outline" size="small" @click="goBack">Cancel</ActionButton>
        <ActionButton size="small" :loading="saving" @click="handleSave">
          {{ isEditing ? 'Update' : 'Create' }}
        </ActionButton>
      </div>
    </div>

    <div v-loading="pageLoading" class="edit-content">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        @submit.prevent
      >
        <ProxyFormLayout v-model="form" :editing="isEditing" />
      </el-form>
    </div>

    <ConfirmDialog
      v-model="leaveDialogVisible"
      title="Unsaved Changes"
      message="You have unsaved changes. Are you sure you want to leave?"
      :is-mobile="isMobile"
      @confirm="handleLeaveConfirm"
      @cancel="handleLeaveCancel"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  type ProxyFormData,
  createDefaultProxyForm,
  formToStoreProxy,
  storeProxyToForm,
} from '../types'
import { getStoreProxy } from '../api/frpc'
import { useProxyStore } from '../stores/proxy'
import ActionButton from '@shared/components/ActionButton.vue'
import ConfirmDialog from '@shared/components/ConfirmDialog.vue'
import ProxyFormLayout from '../components/proxy-form/ProxyFormLayout.vue'
import { useResponsive } from '../composables/useResponsive'

const { isMobile } = useResponsive()
const route = useRoute()
const router = useRouter()
const proxyStore = useProxyStore()

const isEditing = computed(() => !!route.params.name)
const pageLoading = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = ref<ProxyFormData>(createDefaultProxyForm())
const dirty = ref(false)
const formSaved = ref(false)
const trackChanges = ref(false)

const rules: FormRules = {
  name: [
    { required: true, message: 'Name is required', trigger: 'blur' },
    { min: 1, max: 50, message: 'Length should be 1 to 50', trigger: 'blur' },
  ],
  type: [{ required: true, message: 'Type is required', trigger: 'change' }],
  localPort: [
    {
      validator: (_rule, value, callback) => {
        if (!form.value.pluginType && value == null) {
          callback(new Error('Local port is required'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  customDomains: [
    {
      validator: (_rule, value, callback) => {
        if (
          ['http', 'https', 'tcpmux'].includes(form.value.type) &&
          (!value || value.length === 0) &&
          !form.value.subdomain
        ) {
          callback(new Error('Custom domains or subdomain is required'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  healthCheckPath: [
    {
      validator: (_rule, value, callback) => {
        if (form.value.healthCheckType === 'http' && !value) {
          callback(new Error('Path is required for HTTP health check'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

const goBack = () => {
  router.back()
}

watch(
  () => form.value,
  () => {
    if (trackChanges.value) {
      dirty.value = true
    }
  },
  { deep: true },
)

const leaveDialogVisible = ref(false)
const leaveResolve = ref<((value: boolean) => void) | null>(null)

onBeforeRouteLeave(async () => {
  if (dirty.value && !formSaved.value) {
    leaveDialogVisible.value = true
    return new Promise<boolean>((resolve) => {
      leaveResolve.value = resolve
    })
  }
})

const handleLeaveConfirm = () => {
  leaveDialogVisible.value = false
  leaveResolve.value?.(true)
}

const handleLeaveCancel = () => {
  leaveDialogVisible.value = false
  leaveResolve.value?.(false)
}

const loadProxy = async () => {
  const name = route.params.name as string
  if (!name) return

  trackChanges.value = false
  dirty.value = false
  pageLoading.value = true
  try {
    const res = await getStoreProxy(name)
    form.value = storeProxyToForm(res)
    await nextTick()
  } catch (err: any) {
    ElMessage.error('Failed to load proxy: ' + err.message)
    router.push('/proxies?tab=store')
  } finally {
    pageLoading.value = false
    nextTick(() => {
      trackChanges.value = true
    })
  }
}

const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    ElMessage.warning('Please fix the form errors')
    return
  }

  saving.value = true
  try {
    const data = formToStoreProxy(form.value)
    if (isEditing.value) {
      await proxyStore.updateProxy(form.value.name, data)
      ElMessage.success('Proxy updated')
    } else {
      await proxyStore.createProxy(data)
      ElMessage.success('Proxy created')
    }
    formSaved.value = true
    router.push('/proxies?tab=store')
  } catch (err: any) {
    ElMessage.error('Operation failed: ' + (err.message || 'Unknown error'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (isEditing.value) {
    loadProxy()
  } else {
    trackChanges.value = true
  }
})

watch(
  () => route.params.name,
  (name, oldName) => {
    if (name === oldName) return
    if (name) {
      loadProxy()
      return
    }
    trackChanges.value = false
    form.value = createDefaultProxyForm()
    dirty.value = false
    nextTick(() => {
      trackChanges.value = true
    })
  },
)
</script>

<style scoped lang="scss">
.proxy-edit-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 960px;
  margin: 0 auto;
}

/* Edit Header */
.edit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  padding: $spacing-xl 24px;
}

.edit-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 24px 160px;
}

.header-actions {
  display: flex;
  gap: $spacing-sm;
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.breadcrumb-item {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s;
}

.breadcrumb-item:hover {
  color: var(--el-color-primary);
}

.breadcrumb-separator {
  color: var(--el-border-color);
}

.breadcrumb-current {
  color: var(--text-primary);
  font-weight: 500;
}

/* Responsive */
@include mobile {
  .edit-header {
    padding: $spacing-lg;
  }

  .edit-content {
    padding: 0 $spacing-lg 160px;
  }
}
</style>
