import { buildQueryString, http } from './http'
import { formatUnixSeconds } from '../utils/format'
import type { V2Page } from './http'
import type {
  GetProxyResponse,
  ProxyListV2Params,
  ProxyStatsInfo,
  ProxyV2Info,
  ProxyV2Spec,
  ProxyV2SpecBlocks,
  ProxyV2Type,
  TrafficResponse,
} from '../types/proxy'

export interface SystemPruneResponse {
  type: 'offline_proxies'
  cleared: number
  total: number
}

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

const getActiveProxySpec = (
  spec: ProxyV2Spec,
): ProxyV2SpecBlocks[ProxyV2Type] => {
  switch (spec.type) {
    case 'tcp':
      return spec.tcp
    case 'udp':
      return spec.udp
    case 'http':
      return spec.http
    case 'https':
      return spec.https
    case 'tcpmux':
      return spec.tcpmux
    case 'stcp':
      return spec.stcp
    case 'sudp':
      return spec.sudp
    case 'xtcp':
      return spec.xtcp
    default:
      return assertNever(spec)
  }
}

const assertNever = (value: never): never => {
  throw new Error(`Unsupported proxy spec: ${JSON.stringify(value)}`)
}

export const toLegacyProxyStats = (proxy: ProxyV2Info): ProxyStatsInfo => {
  const type = proxy.spec.type
  const activeSpec = getActiveProxySpec(proxy.spec)

  return {
    name: proxy.name,
    type,
    conf: proxy.status.phase === 'offline' ? null : activeSpec,
    user: proxy.user,
    clientID: proxy.clientID,
    todayTrafficIn: proxy.status.todayTrafficIn,
    todayTrafficOut: proxy.status.todayTrafficOut,
    curConns: proxy.status.curConns,
    lastStartTime: formatUnixSeconds(proxy.status.lastStartAt),
    lastCloseTime: formatUnixSeconds(proxy.status.lastCloseAt),
    status: proxy.status.phase,
  }
}

export const getProxy = (type: string, name: string) => {
  return http.get<ProxyStatsInfo>(`../api/proxy/${type}/${name}`)
}

export const getProxyByNameV2 = async (name: string) => {
  const proxy = await http.getV2<ProxyV2Info>(
    `../api/v2/proxies/${encodeURIComponent(name)}`,
  )
  return toLegacyProxyStats(proxy)
}

export const getProxyByName = (name: string) => {
  return http.get<ProxyStatsInfo>(`../api/proxies/${name}`)
}

export const getProxyTraffic = (name: string) => {
  return http.getV2<TrafficResponse>(
    `../api/v2/proxies/${encodeURIComponent(name)}/traffic`,
  )
}

export const clearOfflineProxies = () => {
  return http.postV2<SystemPruneResponse>(
    '../api/v2/system/prune?type=offline_proxies',
  )
}
