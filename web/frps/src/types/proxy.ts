export interface ProxyStatsInfo {
  name: string
  conf: any
  user: string
  clientID: string
  clientVersion: string
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

export interface TrafficResponse {
  name: string
  trafficIn: number[]
  trafficOut: number[]
}
