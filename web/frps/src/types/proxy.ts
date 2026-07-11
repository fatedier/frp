export interface ProxyStatsInfo {
  name: string
  type?: string
  conf: any
  user: string
  clientID: string
  todayTrafficIn: number
  todayTrafficOut: number
  curConns: number
  lastStartTime: string
  lastCloseTime: string
  status: string
}

export interface GetProxyResponse {
  proxies: ProxyStatsInfo[]
}

export interface ProxyListV2Params {
  page?: number
  pageSize?: number
  status?: 'all' | 'online' | 'offline'
  q?: string
  type?: string
  user?: string
  clientID?: string
}

export interface ProxyV2Info {
  name: string
  user: string
  clientID: string
  spec: ProxyV2Spec
  status: ProxyV2Status
}

export interface ProxyV2BaseSpec {
  annotations?: Record<string, string>
  metadatas?: Record<string, string>
  transport?: {
    useEncryption: boolean
    useCompression: boolean
    bandwidthLimit: string
    bandwidthLimitMode: string
  }
  loadBalancer?: {
    group: string
  }
}

export interface ProxyV2TCPBlock extends ProxyV2BaseSpec {
  remotePort?: number
}

export interface ProxyV2UDPBlock extends ProxyV2BaseSpec {
  remotePort?: number
}

export interface ProxyV2HTTPBlock extends ProxyV2BaseSpec {
  customDomains?: string[]
  subdomain?: string
  locations?: string[]
  hostHeaderRewrite?: string
}

export interface ProxyV2HTTPSBlock extends ProxyV2BaseSpec {
  customDomains?: string[]
  subdomain?: string
}

export interface ProxyV2TCPMuxBlock extends ProxyV2BaseSpec {
  customDomains?: string[]
  subdomain?: string
  multiplexer?: string
  routeByHTTPUser?: string
}

export type ProxyV2STCPBlock = ProxyV2BaseSpec

export type ProxyV2SUDPBlock = ProxyV2BaseSpec

export type ProxyV2XTCPBlock = ProxyV2BaseSpec

export interface ProxyV2SpecBlocks {
  tcp: ProxyV2TCPBlock
  udp: ProxyV2UDPBlock
  http: ProxyV2HTTPBlock
  https: ProxyV2HTTPSBlock
  tcpmux: ProxyV2TCPMuxBlock
  stcp: ProxyV2STCPBlock
  sudp: ProxyV2SUDPBlock
  xtcp: ProxyV2XTCPBlock
}

export type ProxyV2Type = keyof ProxyV2SpecBlocks

type ProxyV2SpecFor<T extends ProxyV2Type> = {
  type: T
} & {
  [K in T]: ProxyV2SpecBlocks[K]
} & {
  [K in Exclude<ProxyV2Type, T>]?: never
}

export type ProxyV2Spec = {
  [T in ProxyV2Type]: ProxyV2SpecFor<T>
}[ProxyV2Type]

export interface ProxyV2Status {
  phase: 'online' | 'offline'
  todayTrafficIn: number
  todayTrafficOut: number
  curConns: number
  lastStartAt?: number
  lastCloseAt?: number
}

export interface TrafficResponse {
  name: string
  unit: 'bytes'
  granularity: 'day'
  history: TrafficPoint[]
}

export interface TrafficPoint {
  date: string
  trafficIn: number
  trafficOut: number
}
