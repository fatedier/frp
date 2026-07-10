export interface ClientInfoData {
  key: string
  user: string
  clientID: string
  runID: string
  version?: string
  wireProtocol?: string
  hostname: string
  clientIP?: string
  firstConnectedAt: number
  lastConnectedAt: number
  disconnectedAt?: number
  online: boolean
  status?: ClientStatus
}

export interface ClientStatus {
  phase: 'online' | 'offline'
  curConns: number
  proxyCount: number
}

export interface ClientListV2Params {
  page?: number
  pageSize?: number
  status?: 'all' | 'online' | 'offline'
  q?: string
  user?: string
  clientID?: string
  runID?: string
}
