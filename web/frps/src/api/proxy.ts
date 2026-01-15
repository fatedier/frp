import { http } from './http'
import type { GetProxyResponse, ProxyStatsInfo, TrafficResponse } from '../types/proxy'

export const getProxiesByType = (
  type: string,
  params?: {
    page?: number
    pageSize?: number
  },
) => {
  const searchParams = new URLSearchParams()
  if (params?.page !== undefined) {
    searchParams.set('page', String(params.page))
  }
  if (params?.pageSize !== undefined) {
    searchParams.set('pageSize', String(params.pageSize))
  }

  const query = searchParams.toString()
  return http.get<GetProxyResponse>(`../api/proxy/${type}${query ? `?${query}` : ''}`)
}

export const getProxy = (type: string, name: string) => {
  return http.get<ProxyStatsInfo>(`../api/proxy/${type}/${name}`)
}

export const getProxyTraffic = (name: string) => {
  return http.get<TrafficResponse>(`../api/traffic/${name}`)
}

export const clearOfflineProxies = () => {
  return http.delete('../api/proxies?status=offline')
}
