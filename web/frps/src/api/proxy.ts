import { buildQueryString, http } from './http'
import type { V2Page } from './http'
import type {
  GetProxyResponse,
  ProxyListV2Params,
  ProxyStatsInfo,
  ProxyV2Info,
  TrafficResponse,
} from '../types/proxy'

export const getProxiesByType = (type: string) => {
  return http.get<GetProxyResponse>(`../api/proxy/${type}`)
}

export const getProxiesV2 = async (params: ProxyListV2Params = {}) => {
  const page = await http.getV2<V2Page<ProxyV2Info>>(
    `../api/v2/proxies${buildQueryString({
      page: params.page,
      pageSize: params.pageSize,
      status:
        params.status && params.status !== 'all' ? params.status : undefined,
      q: params.q || undefined,
      type: params.type || undefined,
      user: params.user,
      clientID: params.clientID || undefined,
    })}`,
  )

  return {
    ...page,
    items: page.items.map(toLegacyProxyStats),
  }
}

const toLegacyProxyStats = (proxy: ProxyV2Info): ProxyStatsInfo => ({
  name: proxy.name,
  type: proxy.type,
  conf: proxy.spec,
  user: proxy.user,
  clientID: proxy.clientID,
  todayTrafficIn: proxy.status.todayTrafficIn,
  todayTrafficOut: proxy.status.todayTrafficOut,
  curConns: proxy.status.curConns,
  lastStartTime: proxy.status.lastStartTime,
  lastCloseTime: proxy.status.lastCloseTime,
  status: proxy.status.phase,
})

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
