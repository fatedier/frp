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
  type: string
  user: string
  clientID: string
  spec: any
  status: ProxyV2Status
}

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
