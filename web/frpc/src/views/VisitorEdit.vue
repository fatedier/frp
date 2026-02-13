<template>
  <div class="visitor-edit-page">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
      <a class="breadcrumb-link" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </a>
      <router-link to="/" class="breadcrumb-item">Overview</router-link>
      <span class="breadcrumb-separator">/</span>
      <span class="breadcrumb-current">{{ isEditing ? 'Edit Visitor' : 'Create Visitor' }}</span>
    </nav>

    <div v-loading="pageLoading" class="edit-content">
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-position="top"
        @submit.prevent
      >
        <!-- Header Card -->
        <div class="form-card header-card">
          <div class="card-body">
            <div class="field-row three-col">
              <el-form-item label="Name" prop="name" class="field-grow">
                <el-input
                  v-model="form.name"
                  :disabled="isEditing"
                  placeholder="my-visitor"
                />
              </el-form-item>
              <el-form-item label="Type" prop="type">
                <el-select
                  v-model="form.type"
                  :disabled="isEditing"
                  :fit-input-width="false"
                  popper-class="visitor-type-dropdown"
                  class="type-select"
                >
                  <el-option value="stcp" label="STCP">
                    <div class="type-option">
                      <span class="type-tag-inline type-stcp">STCP</span>
                      <span class="type-desc">Secure TCP Visitor</span>
                    </div>
                  </el-option>
                  <el-option value="sudp" label="SUDP">
                    <div class="type-option">
                      <span class="type-tag-inline type-sudp">SUDP</span>
                      <span class="type-desc">Secure UDP Visitor</span>
                    </div>
                  </el-option>
                  <el-option value="xtcp" label="XTCP">
                    <div class="type-option">
                      <span class="type-tag-inline type-xtcp">XTCP</span>
                      <span class="type-desc">P2P (NAT traversal)</span>
                    </div>
                  </el-option>
                </el-select>
              </el-form-item>
              <el-form-item label="Enabled">
                <el-switch v-model="form.enabled" />
              </el-form-item>
            </div>
          </div>
        </div>

        <!-- Connection -->
        <div class="form-card">
          <div class="card-header">
            <h3 class="card-title">Connection</h3>
          </div>
          <div class="card-body">
            <div class="field-row two-col">
              <el-form-item label="Server Name" prop="serverName">
                <el-input v-model="form.serverName" placeholder="Name of the proxy to visit" />
              </el-form-item>
              <el-form-item label="Server User">
                <el-input v-model="form.serverUser" placeholder="Leave empty for same user" />
              </el-form-item>
            </div>
            <el-form-item label="Secret Key">
              <el-input v-model="form.secretKey" type="password" show-password placeholder="Shared secret" />
            </el-form-item>
            <div class="field-row two-col">
              <el-form-item label="Bind Address">
                <el-input v-model="form.bindAddr" placeholder="127.0.0.1" />
              </el-form-item>
              <el-form-item label="Bind Port" prop="bindPort">
                <el-input-number
                  v-model="form.bindPort"
                  :min="1"
                  :max="65535"
                  controls-position="right"
                  class="full-width"
                />
              </el-form-item>
            </div>
          </div>
        </div>

        <!-- Transport Options (collapsible) -->
        <div class="form-card collapsible-card">
          <div class="card-header clickable" @click="transportExpanded = !transportExpanded">
            <h3 class="card-title">Transport Options</h3>
            <el-icon class="collapse-icon" :class="{ expanded: transportExpanded }"><ArrowDown /></el-icon>
          </div>
          <el-collapse-transition>
            <div v-show="transportExpanded" class="card-body">
              <div class="field-row two-col">
                <el-form-item label="Use Encryption">
                  <el-switch v-model="form.useEncryption" />
                </el-form-item>
                <el-form-item label="Use Compression">
                  <el-switch v-model="form.useCompression" />
                </el-form-item>
              </div>
            </div>
          </el-collapse-transition>
        </div>

        <!-- XTCP Options (collapsible, xtcp only) -->
        <template v-if="form.type === 'xtcp'">
          <div class="form-card collapsible-card">
            <div class="card-header clickable" @click="xtcpExpanded = !xtcpExpanded">
              <h3 class="card-title">XTCP Options</h3>
              <el-icon class="collapse-icon" :class="{ expanded: xtcpExpanded }"><ArrowDown /></el-icon>
            </div>
            <el-collapse-transition>
              <div v-show="xtcpExpanded" class="card-body">
                <el-form-item label="Protocol">
                  <el-select v-model="form.protocol" class="full-width">
                    <el-option value="quic" label="QUIC" />
                    <el-option value="kcp" label="KCP" />
                  </el-select>
                </el-form-item>
                <el-form-item label="Keep Tunnel Open">
                  <el-switch v-model="form.keepTunnelOpen" />
                </el-form-item>
                <div class="field-row two-col">
                  <el-form-item label="Max Retries per Hour">
                    <el-input-number v-model="form.maxRetriesAnHour" :min="0" controls-position="right" class="full-width" />
                  </el-form-item>
                  <el-form-item label="Min Retry Interval (s)">
                    <el-input-number v-model="form.minRetryInterval" :min="0" controls-position="right" class="full-width" />
                  </el-form-item>
                </div>
                <div class="field-row two-col">
                  <el-form-item label="Fallback To">
                    <el-input v-model="form.fallbackTo" placeholder="Fallback visitor name" />
                  </el-form-item>
                  <el-form-item label="Fallback Timeout (ms)">
                    <el-input-number v-model="form.fallbackTimeoutMs" :min="0" controls-position="right" class="full-width" />
                  </el-form-item>
                </div>
              </div>
            </el-collapse-transition>
          </div>

          <!-- NAT Traversal (collapsible, xtcp only) -->
          <div class="form-card collapsible-card">
            <div class="card-header clickable" @click="natExpanded = !natExpanded">
              <h3 class="card-title">NAT Traversal</h3>
              <el-icon class="collapse-icon" :class="{ expanded: natExpanded }"><ArrowDown /></el-icon>
            </div>
            <el-collapse-transition>
              <div v-show="natExpanded" class="card-body">
                <el-form-item label="Disable Assisted Addresses">
                  <el-switch v-model="form.natTraversalDisableAssistedAddrs" />
                  <div class="form-tip">Only use STUN-discovered public addresses</div>
                </el-form-item>
              </div>
            </el-collapse-transition>
          </div>
        </template>
      </el-form>
    </div>

    <!-- Sticky Footer -->
    <div class="sticky-footer">
      <div class="footer-content">
        <el-button @click="goBack">Cancel</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ isEditing ? 'Update' : 'Create' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowDown } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  type VisitorFormData,
  createDefaultVisitorForm,
  formToStoreVisitor,
  storeVisitorToForm,
} from '../types/proxy'
import {
  getStoreVisitor,
  createStoreVisitor,
  updateStoreVisitor,
} from '../api/frpc'

const route = useRoute()
const router = useRouter()

const isEditing = computed(() => !!route.params.name)
const pageLoading = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = ref<VisitorFormData>(createDefaultVisitorForm())

const transportExpanded = ref(false)
const xtcpExpanded = ref(false)
const natExpanded = ref(false)

const formRules: FormRules = {
  name: [
    { required: true, message: 'Name is required', trigger: 'blur' },
    { min: 1, max: 50, message: 'Length should be 1 to 50', trigger: 'blur' },
  ],
  type: [{ required: true, message: 'Type is required', trigger: 'change' }],
  serverName: [
    { required: true, message: 'Server name is required', trigger: 'blur' },
  ],
  bindPort: [
    { required: true, message: 'Bind port is required', trigger: 'blur' },
    { type: 'number', min: 1, message: 'Port must be greater than 0', trigger: 'blur' },
  ],
}

const goBack = () => {
  router.push('/')
}

const loadVisitor = async () => {
  const name = route.params.name as string
  if (!name) return

  pageLoading.value = true
  try {
    const res = await getStoreVisitor(name)
    form.value = storeVisitorToForm(res)
  } catch (err: any) {
    ElMessage.error('Failed to load visitor: ' + err.message)
    router.push('/')
  } finally {
    pageLoading.value = false
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
    const data = formToStoreVisitor(form.value)
    if (isEditing.value) {
      await updateStoreVisitor(form.value.name, data)
      ElMessage.success('Visitor updated')
    } else {
      await createStoreVisitor(data)
      ElMessage.success('Visitor created')
    }
    router.push('/')
  } catch (err: any) {
    ElMessage.error('Operation failed: ' + (err.message || 'Unknown error'))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  if (isEditing.value) {
    loadVisitor()
  }
})
</script>

<style scoped>
.visitor-edit-page {
  padding-bottom: 80px;
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  margin-bottom: 24px;
}

.breadcrumb-link {
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s;
  margin-right: 4px;
}

.breadcrumb-link:hover {
  color: var(--text-primary);
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

/* Form Cards */
.form-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  margin-bottom: 16px;
  overflow: hidden;
}

html.dark .form-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid var(--header-border);
}

html.dark .card-header {
  border-bottom-color: #3a3d5c;
}

.card-header.clickable {
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.card-header.clickable:hover {
  background: var(--hover-bg);
}

.collapsible-card .card-header {
  border-bottom: none;
}

.collapsible-card .card-body {
  border-top: 1px solid var(--header-border);
}

html.dark .collapsible-card .card-body {
  border-top-color: #3a3d5c;
}

.card-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.collapse-icon {
  transition: transform 0.3s;
  color: var(--text-secondary);
}

.collapse-icon.expanded {
  transform: rotate(-180deg);
}

.card-body {
  padding: 20px 24px;
}

/* Field Rows */
.field-row {
  display: grid;
  gap: 16px;
}

.field-row.two-col {
  grid-template-columns: 1fr 1fr;
}

.field-row.three-col {
  grid-template-columns: 1fr auto auto;
  align-items: start;
}

.field-grow {
  min-width: 0;
}

.full-width {
  width: 100%;
}

.type-select {
  width: 180px;
}

.type-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
}

.type-tag-inline {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.5px;
}

.type-tag-inline.type-stcp,
.type-tag-inline.type-sudp,
.type-tag-inline.type-xtcp {
  background: rgba(139, 92, 246, 0.1);
  color: #8b5cf6;
}

.type-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

/* Sticky Footer */
.sticky-footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 99;
  background: var(--header-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-top: 1px solid var(--header-border);
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 16px 40px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* Responsive */
@media (max-width: 768px) {
  .field-row.two-col,
  .field-row.three-col {
    grid-template-columns: 1fr;
  }

  .type-select {
    width: 100%;
  }

  .card-body {
    padding: 16px;
  }

  .footer-content {
    padding: 12px 20px;
  }
}
</style>

<style>
.visitor-type-dropdown {
  min-width: 300px !important;
}

.visitor-type-dropdown .el-select-dropdown__item {
  height: auto;
  padding: 8px 16px;
  line-height: 1.4;
}
</style>
