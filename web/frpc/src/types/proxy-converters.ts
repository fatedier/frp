import type { ProxyType, VisitorType } from './constants'
import type { ProxyFormData, VisitorFormData } from './proxy-form'
import { createDefaultProxyForm, createDefaultVisitorForm } from './proxy-form'
import type { ProxyDefinition, VisitorDefinition } from './proxy-store'

// ========================================
// CONVERTERS: Form -> Store API
// ========================================

export function formToStoreProxy(form: ProxyFormData): ProxyDefinition {
  const block: Record<string, any> = {}

  // Enabled (nil/true = enabled, false = disabled)
  if (!form.enabled) {
    block.enabled = false
  }

  // Backend - LocalIP/LocalPort
  if (form.pluginType === '') {
    if (form.localIP && form.localIP !== '127.0.0.1') {
      block.localIP = form.localIP
    }
    if (form.localPort != null) {
      block.localPort = form.localPort
    }
  } else {
    block.plugin = {
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
    block.transport = {}
    if (form.useEncryption) block.transport.useEncryption = true
    if (form.useCompression) block.transport.useCompression = true
    if (form.bandwidthLimit) block.transport.bandwidthLimit = form.bandwidthLimit
    if (form.bandwidthLimitMode && form.bandwidthLimitMode !== 'client') {
      block.transport.bandwidthLimitMode = form.bandwidthLimitMode
    }
    if (form.proxyProtocolVersion) {
      block.transport.proxyProtocolVersion = form.proxyProtocolVersion
    }
  }

  // Load Balancer
  if (form.loadBalancerGroup) {
    block.loadBalancer = {
      group: form.loadBalancerGroup,
    }
    if (form.loadBalancerGroupKey) {
      block.loadBalancer.groupKey = form.loadBalancerGroupKey
    }
  }

  // Health Check
  if (form.healthCheckType) {
    block.healthCheck = {
      type: form.healthCheckType,
    }
    if (form.healthCheckTimeoutSeconds != null) {
      block.healthCheck.timeoutSeconds = form.healthCheckTimeoutSeconds
    }
    if (form.healthCheckMaxFailed != null) {
      block.healthCheck.maxFailed = form.healthCheckMaxFailed
    }
    if (form.healthCheckIntervalSeconds != null) {
      block.healthCheck.intervalSeconds = form.healthCheckIntervalSeconds
    }
    if (form.healthCheckPath) {
      block.healthCheck.path = form.healthCheckPath
    }
    if (form.healthCheckHTTPHeaders.length > 0) {
      block.healthCheck.httpHeaders = form.healthCheckHTTPHeaders
    }
  }

  // Metadata
  if (form.metadatas.length > 0) {
    block.metadatas = Object.fromEntries(
      form.metadatas.map((m) => [m.key, m.value]),
    )
  }

  // Annotations
  if (form.annotations.length > 0) {
    block.annotations = Object.fromEntries(
      form.annotations.map((a) => [a.key, a.value]),
    )
  }

  // Type-specific fields
  if ((form.type === 'tcp' || form.type === 'udp') && form.remotePort != null) {
    block.remotePort = form.remotePort
  }

  if (form.type === 'http' || form.type === 'https' || form.type === 'tcpmux') {
    if (form.customDomains.length > 0) {
      block.customDomains = form.customDomains.filter(Boolean)
    }
    if (form.subdomain) {
      block.subdomain = form.subdomain
    }
  }

  if (form.type === 'http') {
    if (form.locations.length > 0) {
      block.locations = form.locations.filter(Boolean)
    }
    if (form.httpUser) block.httpUser = form.httpUser
    if (form.httpPassword) block.httpPassword = form.httpPassword
    if (form.hostHeaderRewrite) block.hostHeaderRewrite = form.hostHeaderRewrite
    if (form.routeByHTTPUser) block.routeByHTTPUser = form.routeByHTTPUser

    if (form.requestHeaders.length > 0) {
      block.requestHeaders = {
        set: Object.fromEntries(
          form.requestHeaders.map((h) => [h.key, h.value]),
        ),
      }
    }
    if (form.responseHeaders.length > 0) {
      block.responseHeaders = {
        set: Object.fromEntries(
          form.responseHeaders.map((h) => [h.key, h.value]),
        ),
      }
    }
  }

  if (form.type === 'tcpmux') {
    if (form.httpUser) block.httpUser = form.httpUser
    if (form.httpPassword) block.httpPassword = form.httpPassword
    if (form.routeByHTTPUser) block.routeByHTTPUser = form.routeByHTTPUser
    if (form.multiplexer && form.multiplexer !== 'httpconnect') {
      block.multiplexer = form.multiplexer
    }
  }

  if (form.type === 'stcp' || form.type === 'sudp' || form.type === 'xtcp') {
    if (form.secretKey) block.secretKey = form.secretKey
    if (form.allowUsers.length > 0) {
      block.allowUsers = form.allowUsers.filter(Boolean)
    }
  }

  if (form.type === 'xtcp' && form.natTraversalDisableAssistedAddrs) {
    block.natTraversal = {
      disableAssistedAddrs: true,
    }
  }

  return withStoreProxyBlock(
    {
      name: form.name,
      type: form.type,
    },
    form.type,
    block,
  )
}

export function formToStoreVisitor(form: VisitorFormData): VisitorDefinition {
  const block: Record<string, any> = {}

  if (!form.enabled) {
    block.enabled = false
  }

  if (form.useEncryption || form.useCompression) {
    block.transport = {}
    if (form.useEncryption) block.transport.useEncryption = true
    if (form.useCompression) block.transport.useCompression = true
  }

  if (form.secretKey) block.secretKey = form.secretKey
  if (form.serverUser) block.serverUser = form.serverUser
  if (form.serverName) block.serverName = form.serverName
  if (form.bindAddr && form.bindAddr !== '127.0.0.1') {
    block.bindAddr = form.bindAddr
  }
  if (form.bindPort != null) {
    block.bindPort = form.bindPort
  }

  if (form.type === 'xtcp') {
    if (form.protocol && form.protocol !== 'quic') {
      block.protocol = form.protocol
    }
    if (form.keepTunnelOpen) {
      block.keepTunnelOpen = true
    }
    if (form.maxRetriesAnHour != null) {
      block.maxRetriesAnHour = form.maxRetriesAnHour
    }
    if (form.minRetryInterval != null) {
      block.minRetryInterval = form.minRetryInterval
    }
    if (form.fallbackTo) {
      block.fallbackTo = form.fallbackTo
    }
    if (form.fallbackTimeoutMs != null) {
      block.fallbackTimeoutMs = form.fallbackTimeoutMs
    }
    if (form.natTraversalDisableAssistedAddrs) {
      block.natTraversal = {
        disableAssistedAddrs: true,
      }
    }
  }

  return withStoreVisitorBlock(
    {
      name: form.name,
      type: form.type,
    },
    form.type,
    block,
  )
}

// ========================================
// CONVERTERS: Store API -> Form
// ========================================

function getStoreProxyBlock(config: ProxyDefinition): Record<string, any> {
  switch (config.type) {
    case 'tcp':
      return config.tcp || {}
    case 'udp':
      return config.udp || {}
    case 'http':
      return config.http || {}
    case 'https':
      return config.https || {}
    case 'tcpmux':
      return config.tcpmux || {}
    case 'stcp':
      return config.stcp || {}
    case 'sudp':
      return config.sudp || {}
    case 'xtcp':
      return config.xtcp || {}
  }
}

function withStoreProxyBlock(
  payload: ProxyDefinition,
  type: ProxyType,
  block: Record<string, any>,
): ProxyDefinition {
  switch (type) {
    case 'tcp':
      payload.tcp = block
      break
    case 'udp':
      payload.udp = block
      break
    case 'http':
      payload.http = block
      break
    case 'https':
      payload.https = block
      break
    case 'tcpmux':
      payload.tcpmux = block
      break
    case 'stcp':
      payload.stcp = block
      break
    case 'sudp':
      payload.sudp = block
      break
    case 'xtcp':
      payload.xtcp = block
      break
  }
  return payload
}

function getStoreVisitorBlock(config: VisitorDefinition): Record<string, any> {
  switch (config.type) {
    case 'stcp':
      return config.stcp || {}
    case 'sudp':
      return config.sudp || {}
    case 'xtcp':
      return config.xtcp || {}
  }
}

function withStoreVisitorBlock(
  payload: VisitorDefinition,
  type: VisitorType,
  block: Record<string, any>,
): VisitorDefinition {
  switch (type) {
    case 'stcp':
      payload.stcp = block
      break
    case 'sudp':
      payload.sudp = block
      break
    case 'xtcp':
      payload.xtcp = block
      break
  }
  return payload
}

export function storeProxyToForm(config: ProxyDefinition): ProxyFormData {
  const c = getStoreProxyBlock(config)
  const form = createDefaultProxyForm()

  form.name = config.name || ''
  form.type = config.type || 'tcp'
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
    form.customDomains = c.customDomains
  } else if (c.customDomains) {
    form.customDomains = [c.customDomains]
  }
  form.subdomain = c.subdomain || ''

  // HTTP specific
  if (Array.isArray(c.locations)) {
    form.locations = c.locations
  } else if (c.locations) {
    form.locations = [c.locations]
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
    form.allowUsers = c.allowUsers
  } else if (c.allowUsers) {
    form.allowUsers = [c.allowUsers]
  }

  // XTCP NAT traversal
  form.natTraversalDisableAssistedAddrs =
    c.natTraversal?.disableAssistedAddrs || false

  return form
}

export function storeVisitorToForm(
  config: VisitorDefinition,
): VisitorFormData {
  const c = getStoreVisitorBlock(config)
  const form = createDefaultVisitorForm()

  form.name = config.name || ''
  form.type = config.type || 'stcp'
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
