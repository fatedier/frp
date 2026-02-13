// ========================================
// RUNTIME STATUS TYPES (from /api/status)
// ========================================

export interface ProxyStatus {
  name: string
  type: string
  status: string
  err: string
  local_addr: string
  plugin: string
  remote_addr: string
  source?: 'store' | 'config'
  [key: string]: any
}

export type StatusResponse = Record<string, ProxyStatus[]>

// ========================================
// STORE API TYPES
// ========================================

export interface StoreProxyConfig {
  name: string
  type: string
  config: Record<string, any>
}

export interface StoreVisitorConfig {
  name: string
  type: string
  config: Record<string, any>
}

export interface StoreProxyListResp {
  proxies: StoreProxyConfig[]
}

export interface StoreVisitorListResp {
  visitors: StoreVisitorConfig[]
}

// ========================================
// CONSTANTS
// ========================================

export const PROXY_TYPES = [
  'tcp',
  'udp',
  'http',
  'https',
  'stcp',
  'sudp',
  'xtcp',
  'tcpmux',
] as const

export type ProxyType = (typeof PROXY_TYPES)[number]

export const VISITOR_TYPES = ['stcp', 'sudp', 'xtcp'] as const

export type VisitorType = (typeof VISITOR_TYPES)[number]

export const PLUGIN_TYPES = [
  '',
  'http2https',
  'http_proxy',
  'https2http',
  'https2https',
  'http2http',
  'socks5',
  'static_file',
  'unix_domain_socket',
  'tls2raw',
  'virtual_net',
] as const

export type PluginType = (typeof PLUGIN_TYPES)[number]

// ========================================
// FORM DATA INTERFACES
// ========================================

export interface ProxyFormData {
  // Base fields (ProxyBaseConfig)
  name: string
  type: ProxyType
  enabled: boolean

  // Backend (ProxyBackend)
  localIP: string
  localPort: number | undefined
  pluginType: string
  pluginConfig: Record<string, any>

  // Transport (ProxyTransport)
  useEncryption: boolean
  useCompression: boolean
  bandwidthLimit: string
  bandwidthLimitMode: string
  proxyProtocolVersion: string

  // Load Balancer (LoadBalancerConfig)
  loadBalancerGroup: string
  loadBalancerGroupKey: string

  // Health Check (HealthCheckConfig)
  healthCheckType: string
  healthCheckTimeoutSeconds: number | undefined
  healthCheckMaxFailed: number | undefined
  healthCheckIntervalSeconds: number | undefined
  healthCheckPath: string
  healthCheckHTTPHeaders: Array<{ name: string; value: string }>

  // Metadata & Annotations
  metadatas: Array<{ key: string; value: string }>
  annotations: Array<{ key: string; value: string }>

  // TCP/UDP specific
  remotePort: number | undefined

  // Domain (HTTP/HTTPS/TCPMux) - DomainConfig
  customDomains: string
  subdomain: string

  // HTTP specific (HTTPProxyConfig)
  locations: string
  httpUser: string
  httpPassword: string
  hostHeaderRewrite: string
  requestHeaders: Array<{ key: string; value: string }>
  responseHeaders: Array<{ key: string; value: string }>
  routeByHTTPUser: string

  // TCPMux specific
  multiplexer: string

  // STCP/SUDP/XTCP specific
  secretKey: string
  allowUsers: string

  // XTCP specific (NatTraversalConfig)
  natTraversalDisableAssistedAddrs: boolean
}

export interface VisitorFormData {
  // Base fields (VisitorBaseConfig)
  name: string
  type: VisitorType
  enabled: boolean

  // Transport (VisitorTransport)
  useEncryption: boolean
  useCompression: boolean

  // Connection
  secretKey: string
  serverUser: string
  serverName: string
  bindAddr: string
  bindPort: number | undefined

  // XTCP specific (XTCPVisitorConfig)
  protocol: string
  keepTunnelOpen: boolean
  maxRetriesAnHour: number | undefined
  minRetryInterval: number | undefined
  fallbackTo: string
  fallbackTimeoutMs: number | undefined
  natTraversalDisableAssistedAddrs: boolean
}

// ========================================
// DEFAULT FORM CREATORS
// ========================================

export function createDefaultProxyForm(): ProxyFormData {
  return {
    name: '',
    type: 'tcp',
    enabled: true,

    localIP: '127.0.0.1',
    localPort: undefined,
    pluginType: '',
    pluginConfig: {},

    useEncryption: false,
    useCompression: false,
    bandwidthLimit: '',
    bandwidthLimitMode: 'client',
    proxyProtocolVersion: '',

    loadBalancerGroup: '',
    loadBalancerGroupKey: '',

    healthCheckType: '',
    healthCheckTimeoutSeconds: undefined,
    healthCheckMaxFailed: undefined,
    healthCheckIntervalSeconds: undefined,
    healthCheckPath: '',
    healthCheckHTTPHeaders: [],

    metadatas: [],
    annotations: [],

    remotePort: undefined,

    customDomains: '',
    subdomain: '',

    locations: '',
    httpUser: '',
    httpPassword: '',
    hostHeaderRewrite: '',
    requestHeaders: [],
    responseHeaders: [],
    routeByHTTPUser: '',

    multiplexer: 'httpconnect',

    secretKey: '',
    allowUsers: '',

    natTraversalDisableAssistedAddrs: false,
  }
}

export function createDefaultVisitorForm(): VisitorFormData {
  return {
    name: '',
    type: 'stcp',
    enabled: true,

    useEncryption: false,
    useCompression: false,

    secretKey: '',
    serverUser: '',
    serverName: '',
    bindAddr: '127.0.0.1',
    bindPort: undefined,

    protocol: 'quic',
    keepTunnelOpen: false,
    maxRetriesAnHour: undefined,
    minRetryInterval: undefined,
    fallbackTo: '',
    fallbackTimeoutMs: undefined,
    natTraversalDisableAssistedAddrs: false,
  }
}

// ========================================
// CONVERTERS: Form -> Store API
// ========================================

export function formToStoreProxy(form: ProxyFormData): Record<string, any> {
  const config: Record<string, any> = {
    name: form.name,
    type: form.type,
  }

  // Enabled (nil/true = enabled, false = disabled)
  if (!form.enabled) {
    config.enabled = false
  }

  // Backend - LocalIP/LocalPort
  if (form.pluginType === '') {
    // No plugin, use local backend
    if (form.localIP && form.localIP !== '127.0.0.1') {
      config.localIP = form.localIP
    }
    if (form.localPort != null) {
      config.localPort = form.localPort
    }
  } else {
    // Plugin backend
    config.plugin = {
      type: form.pluginType,
      ...form.pluginConfig,
    }
  }

  // Transport
  if (
    form.useEncryption ||
    form.useCompression ||
    form.bandwidthLimit ||
    (form.bandwidthLimitMode && form.bandwidthLimitMode !== 'client') ||
    form.proxyProtocolVersion
  ) {
    config.transport = {}
    if (form.useEncryption) config.transport.useEncryption = true
    if (form.useCompression) config.transport.useCompression = true
    if (form.bandwidthLimit)
      config.transport.bandwidthLimit = form.bandwidthLimit
    if (form.bandwidthLimitMode && form.bandwidthLimitMode !== 'client') {
      config.transport.bandwidthLimitMode = form.bandwidthLimitMode
    }
    if (form.proxyProtocolVersion) {
      config.transport.proxyProtocolVersion = form.proxyProtocolVersion
    }
  }

  // Load Balancer
  if (form.loadBalancerGroup) {
    config.loadBalancer = {
      group: form.loadBalancerGroup,
    }
    if (form.loadBalancerGroupKey) {
      config.loadBalancer.groupKey = form.loadBalancerGroupKey
    }
  }

  // Health Check
  if (form.healthCheckType) {
    config.healthCheck = {
      type: form.healthCheckType,
    }
    if (form.healthCheckTimeoutSeconds != null) {
      config.healthCheck.timeoutSeconds = form.healthCheckTimeoutSeconds
    }
    if (form.healthCheckMaxFailed != null) {
      config.healthCheck.maxFailed = form.healthCheckMaxFailed
    }
    if (form.healthCheckIntervalSeconds != null) {
      config.healthCheck.intervalSeconds = form.healthCheckIntervalSeconds
    }
    if (form.healthCheckPath) {
      config.healthCheck.path = form.healthCheckPath
    }
    if (form.healthCheckHTTPHeaders.length > 0) {
      config.healthCheck.httpHeaders = form.healthCheckHTTPHeaders
    }
  }

  // Metadata
  if (form.metadatas.length > 0) {
    config.metadatas = Object.fromEntries(
      form.metadatas.map((m) => [m.key, m.value]),
    )
  }

  // Annotations
  if (form.annotations.length > 0) {
    config.annotations = Object.fromEntries(
      form.annotations.map((a) => [a.key, a.value]),
    )
  }

  // Type-specific fields
  if (form.type === 'tcp' || form.type === 'udp') {
    if (form.remotePort != null) {
      config.remotePort = form.remotePort
    }
  }

  if (form.type === 'http' || form.type === 'https' || form.type === 'tcpmux') {
    // Domain config
    if (form.customDomains) {
      config.customDomains = form.customDomains
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    }
    if (form.subdomain) {
      config.subdomain = form.subdomain
    }
  }

  if (form.type === 'http') {
    // HTTP specific
    if (form.locations) {
      config.locations = form.locations
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    }
    if (form.httpUser) config.httpUser = form.httpUser
    if (form.httpPassword) config.httpPassword = form.httpPassword
    if (form.hostHeaderRewrite)
      config.hostHeaderRewrite = form.hostHeaderRewrite
    if (form.routeByHTTPUser) config.routeByHTTPUser = form.routeByHTTPUser

    // Header operations
    if (form.requestHeaders.length > 0) {
      config.requestHeaders = {
        set: Object.fromEntries(
          form.requestHeaders.map((h) => [h.key, h.value]),
        ),
      }
    }
    if (form.responseHeaders.length > 0) {
      config.responseHeaders = {
        set: Object.fromEntries(
          form.responseHeaders.map((h) => [h.key, h.value]),
        ),
      }
    }
  }

  if (form.type === 'tcpmux') {
    // TCPMux specific
    if (form.httpUser) config.httpUser = form.httpUser
    if (form.httpPassword) config.httpPassword = form.httpPassword
    if (form.routeByHTTPUser) config.routeByHTTPUser = form.routeByHTTPUser
    if (form.multiplexer && form.multiplexer !== 'httpconnect') {
      config.multiplexer = form.multiplexer
    }
  }

  if (form.type === 'stcp' || form.type === 'sudp' || form.type === 'xtcp') {
    // Secure proxy types
    if (form.secretKey) config.secretKey = form.secretKey
    if (form.allowUsers) {
      config.allowUsers = form.allowUsers
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    }
  }

  if (form.type === 'xtcp') {
    // XTCP NAT traversal
    if (form.natTraversalDisableAssistedAddrs) {
      config.natTraversal = {
        disableAssistedAddrs: true,
      }
    }
  }

  return config
}

export function formToStoreVisitor(form: VisitorFormData): Record<string, any> {
  const config: Record<string, any> = {
    name: form.name,
    type: form.type,
  }

  // Enabled
  if (!form.enabled) {
    config.enabled = false
  }

  // Transport
  if (form.useEncryption || form.useCompression) {
    config.transport = {}
    if (form.useEncryption) config.transport.useEncryption = true
    if (form.useCompression) config.transport.useCompression = true
  }

  // Base fields
  if (form.secretKey) config.secretKey = form.secretKey
  if (form.serverUser) config.serverUser = form.serverUser
  if (form.serverName) config.serverName = form.serverName
  if (form.bindAddr && form.bindAddr !== '127.0.0.1') {
    config.bindAddr = form.bindAddr
  }
  if (form.bindPort != null) {
    config.bindPort = form.bindPort
  }

  // XTCP specific
  if (form.type === 'xtcp') {
    if (form.protocol && form.protocol !== 'quic') {
      config.protocol = form.protocol
    }
    if (form.keepTunnelOpen) {
      config.keepTunnelOpen = true
    }
    if (form.maxRetriesAnHour != null) {
      config.maxRetriesAnHour = form.maxRetriesAnHour
    }
    if (form.minRetryInterval != null) {
      config.minRetryInterval = form.minRetryInterval
    }
    if (form.fallbackTo) {
      config.fallbackTo = form.fallbackTo
    }
    if (form.fallbackTimeoutMs != null) {
      config.fallbackTimeoutMs = form.fallbackTimeoutMs
    }
    if (form.natTraversalDisableAssistedAddrs) {
      config.natTraversal = {
        disableAssistedAddrs: true,
      }
    }
  }

  return config
}

// ========================================
// CONVERTERS: Store API -> Form
// ========================================

export function storeProxyToForm(config: StoreProxyConfig): ProxyFormData {
  const c = config.config || {}
  const form = createDefaultProxyForm()

  form.name = config.name || ''
  form.type = (config.type as ProxyType) || 'tcp'
  form.enabled = c.enabled !== false

  // Backend
  form.localIP = c.localIP || '127.0.0.1'
  form.localPort = c.localPort
  if (c.plugin?.type) {
    form.pluginType = c.plugin.type
    form.pluginConfig = { ...c.plugin }
    delete form.pluginConfig.type
  }

  // Transport
  if (c.transport) {
    form.useEncryption = c.transport.useEncryption || false
    form.useCompression = c.transport.useCompression || false
    form.bandwidthLimit = c.transport.bandwidthLimit || ''
    form.bandwidthLimitMode = c.transport.bandwidthLimitMode || 'client'
    form.proxyProtocolVersion = c.transport.proxyProtocolVersion || ''
  }

  // Load Balancer
  if (c.loadBalancer) {
    form.loadBalancerGroup = c.loadBalancer.group || ''
    form.loadBalancerGroupKey = c.loadBalancer.groupKey || ''
  }

  // Health Check
  if (c.healthCheck) {
    form.healthCheckType = c.healthCheck.type || ''
    form.healthCheckTimeoutSeconds = c.healthCheck.timeoutSeconds
    form.healthCheckMaxFailed = c.healthCheck.maxFailed
    form.healthCheckIntervalSeconds = c.healthCheck.intervalSeconds
    form.healthCheckPath = c.healthCheck.path || ''
    form.healthCheckHTTPHeaders = c.healthCheck.httpHeaders || []
  }

  // Metadata
  if (c.metadatas) {
    form.metadatas = Object.entries(c.metadatas).map(([key, value]) => ({
      key,
      value: String(value),
    }))
  }

  // Annotations
  if (c.annotations) {
    form.annotations = Object.entries(c.annotations).map(([key, value]) => ({
      key,
      value: String(value),
    }))
  }

  // Type-specific fields
  form.remotePort = c.remotePort

  // Domain config
  if (Array.isArray(c.customDomains)) {
    form.customDomains = c.customDomains.join(', ')
  } else if (c.customDomains) {
    form.customDomains = c.customDomains
  }
  form.subdomain = c.subdomain || ''

  // HTTP specific
  if (Array.isArray(c.locations)) {
    form.locations = c.locations.join(', ')
  } else if (c.locations) {
    form.locations = c.locations
  }
  form.httpUser = c.httpUser || ''
  form.httpPassword = c.httpPassword || ''
  form.hostHeaderRewrite = c.hostHeaderRewrite || ''
  form.routeByHTTPUser = c.routeByHTTPUser || ''

  // Header operations
  if (c.requestHeaders?.set) {
    form.requestHeaders = Object.entries(c.requestHeaders.set).map(
      ([key, value]) => ({ key, value: String(value) }),
    )
  }
  if (c.responseHeaders?.set) {
    form.responseHeaders = Object.entries(c.responseHeaders.set).map(
      ([key, value]) => ({ key, value: String(value) }),
    )
  }

  // TCPMux
  form.multiplexer = c.multiplexer || 'httpconnect'

  // Secure types
  form.secretKey = c.secretKey || ''
  if (Array.isArray(c.allowUsers)) {
    form.allowUsers = c.allowUsers.join(', ')
  } else if (c.allowUsers) {
    form.allowUsers = c.allowUsers
  }

  // XTCP NAT traversal
  form.natTraversalDisableAssistedAddrs =
    c.natTraversal?.disableAssistedAddrs || false

  return form
}

export function storeVisitorToForm(
  config: StoreVisitorConfig,
): VisitorFormData {
  const c = config.config || {}
  const form = createDefaultVisitorForm()

  form.name = config.name || ''
  form.type = (config.type as VisitorType) || 'stcp'
  form.enabled = c.enabled !== false

  // Transport
  if (c.transport) {
    form.useEncryption = c.transport.useEncryption || false
    form.useCompression = c.transport.useCompression || false
  }

  // Base fields
  form.secretKey = c.secretKey || ''
  form.serverUser = c.serverUser || ''
  form.serverName = c.serverName || ''
  form.bindAddr = c.bindAddr || '127.0.0.1'
  form.bindPort = c.bindPort

  // XTCP specific
  form.protocol = c.protocol || 'quic'
  form.keepTunnelOpen = c.keepTunnelOpen || false
  form.maxRetriesAnHour = c.maxRetriesAnHour
  form.minRetryInterval = c.minRetryInterval
  form.fallbackTo = c.fallbackTo || ''
  form.fallbackTimeoutMs = c.fallbackTimeoutMs
  form.natTraversalDisableAssistedAddrs =
    c.natTraversal?.disableAssistedAddrs || false

  return form
}
