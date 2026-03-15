import type { ProxyType, VisitorType } from './constants'

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
  customDomains: string[]
  subdomain: string

  // HTTP specific (HTTPProxyConfig)
  locations: string[]
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
  allowUsers: string[]

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

    customDomains: [],
    subdomain: '',

    locations: [],
    httpUser: '',
    httpPassword: '',
    hostHeaderRewrite: '',
    requestHeaders: [],
    responseHeaders: [],
    routeByHTTPUser: '',

    multiplexer: 'httpconnect',

    secretKey: '',
    allowUsers: [],

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
