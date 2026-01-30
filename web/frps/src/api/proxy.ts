import { http } from './http'
import type {
  GetProxyResponse,
  ProxyStatsInfo,
  TrafficResponse,
} from '../types/proxy'

export const getProxiesByType = (type: string) => {
  return http.get<GetProxyResponse>(`../api/proxy/${type}`)
}

export const getProxy = (type: string, name: string) => {
  return http.get<ProxyStatsInfo>(`../api/proxy/${type}/${name}`)
}

export const getProxyByName = (name: string) => {
  return http.get<ProxyStatsInfo>(`../api/proxies/${name}`)
}

export const getProxyTraffic = (name: string) => {
  return http.get<TrafficResponse>(`../api/traffic/${name}`)
}

export const clearOfflineProxies = () => {
  return http.delete('../api/proxies?status=offline')
}
