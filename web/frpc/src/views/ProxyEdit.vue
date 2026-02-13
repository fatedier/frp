<template>
  <div class="proxy-edit-page">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
      <a class="breadcrumb-link" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </a>
      <router-link to="/" class="breadcrumb-item">Overview</router-link>
      <span class="breadcrumb-separator">/</span>
      <span class="breadcrumb-current">{{ isEditing ? 'Edit Proxy' : 'Create Proxy' }}</span>
    </nav>

    <div v-loading="pageLoading" class="edit-content">
      <div class="two-column-layout">
        <!-- Left: Section Nav (desktop only) -->
        <aside class="section-nav">
          <div class="nav-sticky">
            <div class="nav-title">Sections</div>
            <a
              v-for="section in visibleSections"
              :key="section.id"
              class="nav-item"
              :class="{ active: activeSection === section.id }"
              @click="scrollToSection(section.id)"
            >
              {{ section.label }}
            </a>
          </div>
        </aside>

        <!-- Right: Form -->
        <div class="form-area">
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            @submit.prevent
          >
            <!-- Header Card -->
            <div id="section-basic" class="form-card header-card">
              <div class="card-body">
                <div class="field-row three-col">
                  <el-form-item label="Name" prop="name" class="field-grow">
                    <el-input
                      v-model="form.name"
                      :disabled="isEditing"
                      placeholder="my-proxy"
                    />
                  </el-form-item>
                  <el-form-item label="Type" prop="type">
                    <el-select
                      v-model="form.type"
                      :disabled="isEditing"
                      :fit-input-width="false"
                      popper-class="type-dropdown"
                      class="type-select"
                    >
                      <el-option
                        v-for="t in PROXY_TYPES"
                        :key="t"
                        :label="t.toUpperCase()"
                        :value="t"
                      >
                        <div class="type-option">
                          <span class="type-tag-inline" :class="`type-${t}`">{{ t.toUpperCase() }}</span>
                          <span class="type-desc">{{ typeDescs[t] }}</span>
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

            <!-- Backend Service -->
            <div id="section-backend" class="form-card">
              <div class="card-header">
                <h3 class="card-title">Backend Service</h3>
              </div>
              <div class="card-body">
                <el-form-item label="Backend Mode">
                  <el-radio-group v-model="backendMode">
                    <el-radio value="direct">Direct</el-radio>
                    <el-radio value="plugin">Plugin</el-radio>
                  </el-radio-group>
                </el-form-item>

                <template v-if="backendMode === 'direct'">
                  <div class="field-row two-col">
                    <el-form-item label="Local IP" prop="localIP">
                      <el-input v-model="form.localIP" placeholder="127.0.0.1" />
                    </el-form-item>
                    <el-form-item label="Local Port" prop="localPort">
                      <el-input-number
                        v-model="form.localPort"
                        :min="0"
                        :max="65535"
                        placeholder="80"
                        controls-position="right"
                        class="full-width"
                      />
                    </el-form-item>
                  </div>
                </template>

                <template v-else>
                  <el-form-item label="Plugin Type">
                    <el-select v-model="form.pluginType" class="full-width">
                      <el-option
                        v-for="p in PLUGIN_LIST"
                        :key="p"
                        :label="p"
                        :value="p"
                      />
                    </el-select>
                  </el-form-item>

                  <!-- Plugin-specific fields -->
                  <template v-if="['http2https', 'https2http', 'https2https', 'http2http', 'tls2raw'].includes(form.pluginType)">
                    <el-form-item label="Local Address">
                      <el-input v-model="form.pluginConfig.localAddr" placeholder="127.0.0.1:8080" />
                    </el-form-item>
                  </template>
                  <template v-if="['http2https', 'https2http', 'https2https', 'http2http'].includes(form.pluginType)">
                    <el-form-item label="Host Header Rewrite">
                      <el-input v-model="form.pluginConfig.hostHeaderRewrite" />
                    </el-form-item>
                    <el-form-item label="Request Headers">
                      <KeyValueEditor v-model="pluginRequestHeaders" key-placeholder="Header" value-placeholder="Value" />
                    </el-form-item>
                  </template>
                  <template v-if="['https2http', 'https2https'].includes(form.pluginType)">
                    <el-form-item label="Enable HTTP/2">
                      <el-switch v-model="form.pluginConfig.enableHTTP2" />
                    </el-form-item>
                    <div class="field-row two-col">
                      <el-form-item label="Certificate Path">
                        <el-input v-model="form.pluginConfig.crtPath" placeholder="/path/to/cert.pem" />
                      </el-form-item>
                      <el-form-item label="Key Path">
                        <el-input v-model="form.pluginConfig.keyPath" placeholder="/path/to/key.pem" />
                      </el-form-item>
                    </div>
                  </template>
                  <template v-if="form.pluginType === 'tls2raw'">
                    <div class="field-row two-col">
                      <el-form-item label="Certificate Path">
                        <el-input v-model="form.pluginConfig.crtPath" placeholder="/path/to/cert.pem" />
                      </el-form-item>
                      <el-form-item label="Key Path">
                        <el-input v-model="form.pluginConfig.keyPath" placeholder="/path/to/key.pem" />
                      </el-form-item>
                    </div>
                  </template>
                  <template v-if="form.pluginType === 'http_proxy'">
                    <div class="field-row two-col">
                      <el-form-item label="HTTP User">
                        <el-input v-model="form.pluginConfig.httpUser" />
                      </el-form-item>
                      <el-form-item label="HTTP Password">
                        <el-input v-model="form.pluginConfig.httpPassword" type="password" show-password />
                      </el-form-item>
                    </div>
                  </template>
                  <template v-if="form.pluginType === 'socks5'">
                    <div class="field-row two-col">
                      <el-form-item label="Username">
                        <el-input v-model="form.pluginConfig.username" />
                      </el-form-item>
                      <el-form-item label="Password">
                        <el-input v-model="form.pluginConfig.password" type="password" show-password />
                      </el-form-item>
                    </div>
                  </template>
                  <template v-if="form.pluginType === 'static_file'">
                    <el-form-item label="Local Path">
                      <el-input v-model="form.pluginConfig.localPath" placeholder="/path/to/files" />
                    </el-form-item>
                    <el-form-item label="Strip Prefix">
                      <el-input v-model="form.pluginConfig.stripPrefix" />
                    </el-form-item>
                    <div class="field-row two-col">
                      <el-form-item label="HTTP User">
                        <el-input v-model="form.pluginConfig.httpUser" />
                      </el-form-item>
                      <el-form-item label="HTTP Password">
                        <el-input v-model="form.pluginConfig.httpPassword" type="password" show-password />
                      </el-form-item>
                    </div>
                  </template>
                  <template v-if="form.pluginType === 'unix_domain_socket'">
                    <el-form-item label="Unix Socket Path">
                      <el-input v-model="form.pluginConfig.unixPath" placeholder="/tmp/socket.sock" />
                    </el-form-item>
                  </template>
                </template>
              </div>
            </div>

            <!-- Remote Configuration -->
            <div
              v-if="['tcp', 'udp', 'http', 'https', 'tcpmux'].includes(form.type)"
              id="section-remote"
              class="form-card"
            >
              <div class="card-header">
                <h3 class="card-title">Remote Configuration</h3>
              </div>
              <div class="card-body">
                <template v-if="['tcp', 'udp'].includes(form.type)">
                  <el-form-item label="Remote Port" prop="remotePort">
                    <el-input-number
                      v-model="form.remotePort"
                      :min="0"
                      :max="65535"
                      controls-position="right"
                      class="full-width"
                    />
                    <div class="form-tip">Use 0 for random port assignment</div>
                  </el-form-item>
                </template>
                <template v-if="['http', 'https', 'tcpmux'].includes(form.type)">
                  <el-form-item label="Custom Domains" prop="customDomains">
                    <el-input v-model="form.customDomains" placeholder="example.com, www.example.com" />
                    <div class="form-tip">Comma-separated list of domains</div>
                  </el-form-item>
                  <el-form-item v-if="form.type !== 'tcpmux'" label="Subdomain">
                    <el-input v-model="form.subdomain" placeholder="test" />
                  </el-form-item>
                  <el-form-item v-if="form.type === 'tcpmux'" label="Multiplexer">
                    <el-select v-model="form.multiplexer" class="full-width">
                      <el-option label="HTTP CONNECT" value="httpconnect" />
                    </el-select>
                  </el-form-item>
                </template>
              </div>
            </div>

            <!-- Authentication -->
            <div
              v-if="['http', 'tcpmux', 'stcp', 'sudp', 'xtcp'].includes(form.type)"
              id="section-auth"
              class="form-card"
            >
              <div class="card-header">
                <h3 class="card-title">Authentication</h3>
              </div>
              <div class="card-body">
                <template v-if="['http', 'tcpmux'].includes(form.type)">
                  <div class="field-row two-col">
                    <el-form-item label="HTTP User">
                      <el-input v-model="form.httpUser" />
                    </el-form-item>
                    <el-form-item label="HTTP Password">
                      <el-input v-model="form.httpPassword" type="password" show-password />
                    </el-form-item>
                  </div>
                  <el-form-item label="Route By HTTP User">
                    <el-input v-model="form.routeByHTTPUser" />
                  </el-form-item>
                </template>
                <template v-if="['stcp', 'sudp', 'xtcp'].includes(form.type)">
                  <el-form-item label="Secret Key" prop="secretKey">
                    <el-input v-model="form.secretKey" type="password" show-password />
                  </el-form-item>
                  <el-form-item label="Allow Users">
                    <el-input v-model="form.allowUsers" placeholder="user1, user2" />
                    <div class="form-tip">Comma-separated list of allowed users</div>
                  </el-form-item>
                </template>
              </div>
            </div>

            <!-- HTTP Options (http type only) -->
            <div
              v-if="form.type === 'http'"
              id="section-http"
              class="form-card collapsible-card"
            >
              <div class="card-header clickable" @click="sections.httpOptions = !sections.httpOptions">
                <h3 class="card-title">HTTP Options</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.httpOptions }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.httpOptions" class="card-body">
                  <el-form-item label="Locations">
                    <el-input v-model="form.locations" placeholder="/path1, /path2" />
                    <div class="form-tip">Comma-separated URL paths</div>
                  </el-form-item>
                  <el-form-item label="Host Header Rewrite">
                    <el-input v-model="form.hostHeaderRewrite" />
                  </el-form-item>
                  <el-form-item label="Request Headers">
                    <KeyValueEditor v-model="form.requestHeaders" key-placeholder="Header" value-placeholder="Value" />
                  </el-form-item>
                  <el-form-item label="Response Headers">
                    <KeyValueEditor v-model="form.responseHeaders" key-placeholder="Header" value-placeholder="Value" />
                  </el-form-item>
                </div>
              </el-collapse-transition>
            </div>

            <!-- Transport -->
            <div id="section-transport" class="form-card collapsible-card">
              <div class="card-header clickable" @click="sections.transport = !sections.transport">
                <h3 class="card-title">Transport</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.transport }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.transport" class="card-body">
                  <div class="field-row two-col">
                    <el-form-item label="Use Encryption">
                      <el-switch v-model="form.useEncryption" />
                    </el-form-item>
                    <el-form-item label="Use Compression">
                      <el-switch v-model="form.useCompression" />
                    </el-form-item>
                  </div>
                  <div class="field-row two-col">
                    <el-form-item label="Bandwidth Limit">
                      <el-input v-model="form.bandwidthLimit" placeholder="1MB" />
                      <div class="form-tip">e.g., 1MB, 500KB</div>
                    </el-form-item>
                    <el-form-item label="Bandwidth Limit Mode">
                      <el-select v-model="form.bandwidthLimitMode" class="full-width">
                        <el-option label="Client" value="client" />
                        <el-option label="Server" value="server" />
                      </el-select>
                    </el-form-item>
                  </div>
                  <el-form-item label="Proxy Protocol Version">
                    <el-select v-model="form.proxyProtocolVersion" class="full-width">
                      <el-option label="None" value="" />
                      <el-option label="v1" value="v1" />
                      <el-option label="v2" value="v2" />
                    </el-select>
                  </el-form-item>
                </div>
              </el-collapse-transition>
            </div>

            <!-- Health Check -->
            <div id="section-health" class="form-card collapsible-card">
              <div class="card-header clickable" @click="sections.healthCheck = !sections.healthCheck">
                <h3 class="card-title">Health Check</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.healthCheck }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.healthCheck" class="card-body">
                  <el-form-item label="Type">
                    <el-select v-model="form.healthCheckType" class="full-width">
                      <el-option label="Disabled" value="" />
                      <el-option label="TCP" value="tcp" />
                      <el-option label="HTTP" value="http" />
                    </el-select>
                  </el-form-item>
                  <template v-if="form.healthCheckType">
                    <div class="field-row three-col">
                      <el-form-item label="Timeout (s)">
                        <el-input-number v-model="form.healthCheckTimeoutSeconds" :min="1" controls-position="right" class="full-width" />
                      </el-form-item>
                      <el-form-item label="Max Failed">
                        <el-input-number v-model="form.healthCheckMaxFailed" :min="1" controls-position="right" class="full-width" />
                      </el-form-item>
                      <el-form-item label="Interval (s)">
                        <el-input-number v-model="form.healthCheckIntervalSeconds" :min="1" controls-position="right" class="full-width" />
                      </el-form-item>
                    </div>
                    <template v-if="form.healthCheckType === 'http'">
                      <el-form-item label="Path" prop="healthCheckPath">
                        <el-input v-model="form.healthCheckPath" placeholder="/health" />
                      </el-form-item>
                      <el-form-item label="HTTP Headers">
                        <KeyValueEditor v-model="healthCheckHeaders" key-placeholder="Header" value-placeholder="Value" />
                      </el-form-item>
                    </template>
                  </template>
                </div>
              </el-collapse-transition>
            </div>

            <!-- Load Balancer -->
            <div id="section-lb" class="form-card collapsible-card">
              <div class="card-header clickable" @click="sections.loadBalancer = !sections.loadBalancer">
                <h3 class="card-title">Load Balancer</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.loadBalancer }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.loadBalancer" class="card-body">
                  <div class="field-row two-col">
                    <el-form-item label="Group">
                      <el-input v-model="form.loadBalancerGroup" placeholder="Group name" />
                    </el-form-item>
                    <el-form-item label="Group Key">
                      <el-input v-model="form.loadBalancerGroupKey" />
                    </el-form-item>
                  </div>
                </div>
              </el-collapse-transition>
            </div>

            <!-- NAT Traversal (XTCP only) -->
            <div
              v-if="form.type === 'xtcp'"
              id="section-nat"
              class="form-card collapsible-card"
            >
              <div class="card-header clickable" @click="sections.natTraversal = !sections.natTraversal">
                <h3 class="card-title">NAT Traversal</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.natTraversal }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.natTraversal" class="card-body">
                  <el-form-item label="Disable Assisted Addresses">
                    <el-switch v-model="form.natTraversalDisableAssistedAddrs" />
                    <div class="form-tip">Only use STUN-discovered public addresses</div>
                  </el-form-item>
                </div>
              </el-collapse-transition>
            </div>

            <!-- Metadata & Annotations -->
            <div id="section-meta" class="form-card collapsible-card">
              <div class="card-header clickable" @click="sections.metadata = !sections.metadata">
                <h3 class="card-title">Metadata & Annotations</h3>
                <el-icon class="collapse-icon" :class="{ expanded: sections.metadata }"><ArrowDown /></el-icon>
              </div>
              <el-collapse-transition>
                <div v-show="sections.metadata" class="card-body">
                  <el-form-item label="Metadatas">
                    <KeyValueEditor v-model="form.metadatas" />
                  </el-form-item>
                  <el-form-item label="Annotations">
                    <KeyValueEditor v-model="form.annotations" />
                  </el-form-item>
                </div>
              </el-collapse-transition>
            </div>
          </el-form>
        </div>
      </div>
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
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowDown } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  PROXY_TYPES,
  type ProxyFormData,
  createDefaultProxyForm,
  formToStoreProxy,
  storeProxyToForm,
} from '../types/proxy'
import {
  getStoreProxy,
  createStoreProxy,
  updateStoreProxy,
} from '../api/frpc'
import KeyValueEditor from '../components/KeyValueEditor.vue'

const route = useRoute()
const router = useRouter()

const isEditing = computed(() => !!route.params.name)
const pageLoading = ref(false)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = ref<ProxyFormData>(createDefaultProxyForm())
const backendMode = ref<'direct' | 'plugin'>('direct')
const activeSection = ref('section-basic')

const PLUGIN_LIST = [
  'http2https',
  'http_proxy',
  'https2http',
  'https2https',
  'http2http',
  'socks5',
  'static_file',
  'unix_domain_socket',
  'tls2raw',
]

const sections = reactive({
  httpOptions: false,
  transport: false,
  healthCheck: false,
  loadBalancer: false,
  natTraversal: false,
  metadata: false,
})

const typeDescs: Record<string, string> = {
  tcp: 'TCP port forwarding',
  udp: 'UDP port forwarding',
  http: 'HTTP virtual host routing',
  https: 'HTTPS virtual host routing',
  stcp: 'Secure TCP with visitor auth',
  sudp: 'Secure UDP with visitor auth',
  xtcp: 'P2P through NAT traversal',
  tcpmux: 'TCP multiplexing (HTTP CONNECT)',
}

// Plugin request headers as KV array (synced with pluginConfig)
const pluginRequestHeaders = computed({
  get() {
    const set = form.value.pluginConfig?.requestHeaders?.set
    if (!set || typeof set !== 'object') return []
    return Object.entries(set).map(([key, value]) => ({ key, value: String(value) }))
  },
  set(val: Array<{ key: string; value: string }>) {
    if (!form.value.pluginConfig) form.value.pluginConfig = {}
    if (val.length === 0) {
      delete form.value.pluginConfig.requestHeaders
    } else {
      form.value.pluginConfig.requestHeaders = {
        set: Object.fromEntries(val.map((e) => [e.key, e.value])),
      }
    }
  },
})

// Health check HTTP headers adapter ({ name, value } <-> { key, value })
const healthCheckHeaders = computed({
  get() {
    return form.value.healthCheckHTTPHeaders.map((h) => ({ key: h.name, value: h.value }))
  },
  set(val: Array<{ key: string; value: string }>) {
    form.value.healthCheckHTTPHeaders = val.map((h) => ({ name: h.key, value: h.value }))
  },
})

const allSections = [
  { id: 'section-basic', label: 'Basic', always: true },
  { id: 'section-backend', label: 'Backend', always: true },
  { id: 'section-remote', label: 'Remote', types: ['tcp', 'udp', 'http', 'https', 'tcpmux'] },
  { id: 'section-auth', label: 'Auth', types: ['http', 'tcpmux', 'stcp', 'sudp', 'xtcp'] },
  { id: 'section-http', label: 'HTTP', types: ['http'] },
  { id: 'section-transport', label: 'Transport', always: true },
  { id: 'section-health', label: 'Health', always: true },
  { id: 'section-lb', label: 'Load Balancer', always: true },
  { id: 'section-nat', label: 'NAT', types: ['xtcp'] },
  { id: 'section-meta', label: 'Metadata', always: true },
]

const visibleSections = computed(() => {
  return allSections.filter(
    (s) => s.always || (s.types && s.types.includes(form.value.type)),
  )
})

const rules: FormRules = {
  name: [
    { required: true, message: 'Name is required', trigger: 'blur' },
    { min: 1, max: 50, message: 'Length should be 1 to 50', trigger: 'blur' },
  ],
  type: [{ required: true, message: 'Type is required', trigger: 'change' }],
  localPort: [
    {
      validator: (_rule, value, callback) => {
        if (backendMode.value === 'direct' && value == null) {
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
          !value &&
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

const scrollToSection = (id: string) => {
  activeSection.value = id
  const el = document.getElementById(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

const goBack = () => {
  router.push('/')
}

// Reset plugin config when plugin type changes
watch(
  () => form.value.pluginType,
  (newType, oldType) => {
    if (newType !== oldType) {
      // Preserve type, reset the rest
      const preserved = form.value.pluginConfig
      if (preserved && Object.keys(preserved).length > 0) {
        // Only reset if type actually changed from a different plugin
        form.value.pluginConfig = {}
      }
    }
  },
)

// Sync backendMode with form plugin state
watch(backendMode, (mode) => {
  if (mode === 'direct') {
    form.value.pluginType = ''
    form.value.pluginConfig = {}
  } else if (!form.value.pluginType) {
    form.value.pluginType = 'http2https'
  }
})

const loadProxy = async () => {
  const name = route.params.name as string
  if (!name) return

  pageLoading.value = true
  try {
    const res = await getStoreProxy(name)
    form.value = storeProxyToForm(res)
    if (form.value.pluginType) {
      backendMode.value = 'plugin'
    }
  } catch (err: any) {
    ElMessage.error('Failed to load proxy: ' + err.message)
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
    const data = formToStoreProxy(form.value)
    if (isEditing.value) {
      await updateStoreProxy(form.value.name, data)
      ElMessage.success('Proxy updated')
    } else {
      await createStoreProxy(data)
      ElMessage.success('Proxy created')
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
    loadProxy()
  }
})
</script>

<style scoped>
.proxy-edit-page {
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

/* Two-column Layout */
.two-column-layout {
  display: flex;
  gap: 24px;
}

.section-nav {
  width: 160px;
  flex-shrink: 0;
}

.nav-sticky {
  position: sticky;
  top: 130px;
}

.nav-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
  margin-bottom: 12px;
  padding-left: 12px;
}

.nav-item {
  display: block;
  padding: 8px 12px;
  font-size: 13px;
  color: var(--text-secondary);
  text-decoration: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  margin-bottom: 2px;
}

.nav-item:hover {
  color: var(--text-primary);
  background: var(--hover-bg);
}

.nav-item.active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  font-weight: 500;
}

.form-area {
  flex: 1;
  min-width: 0;
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

.header-card .card-body {
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

/* Type option in dropdown */
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
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}

.type-tag-inline.type-tcp { background: rgba(59, 130, 246, 0.1); color: #3b82f6; }
.type-tag-inline.type-udp { background: rgba(245, 158, 11, 0.1); color: #f59e0b; }
.type-tag-inline.type-http { background: rgba(16, 185, 129, 0.1); color: #10b981; }
.type-tag-inline.type-https { background: rgba(16, 185, 129, 0.15); color: #059669; }
.type-tag-inline.type-stcp,
.type-tag-inline.type-sudp,
.type-tag-inline.type-xtcp { background: rgba(139, 92, 246, 0.1); color: #8b5cf6; }
.type-tag-inline.type-tcpmux { background: rgba(236, 72, 153, 0.1); color: #ec4899; }

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
  .section-nav {
    display: none;
  }

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
/* Global: type dropdown popper needs to be wider than the input */
.type-dropdown {
  min-width: 320px !important;
}

.type-dropdown .el-select-dropdown__item {
  height: auto;
  padding: 8px 16px;
  line-height: 1.4;
}
</style>
