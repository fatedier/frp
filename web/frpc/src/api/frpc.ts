import { http } from './http'
import type {
  StatusResponse,
  ProxyListResp,
  ProxyDefinition,
  VisitorListResp,
  VisitorDefinition,
} from '../types'

export const getStatus = () => {
  return http.get<StatusResponse>('/api/status')
}

export const getConfig = () => {
  return http.get<string>('/api/config')
}

export const putConfig = (content: string) => {
  return http.put<void>('/api/config', content)
}

export const reloadConfig = () => {
  return http.get<void>('/api/reload')
}

// Config lookup API (any source)
export const getProxyConfig = (name: string) => {
  return http.get<ProxyDefinition>(
    `/api/proxy/${encodeURIComponent(name)}/config`,
  )
}

export const getVisitorConfig = (name: string) => {
  return http.get<VisitorDefinition>(
    `/api/visitor/${encodeURIComponent(name)}/config`,
  )
}

// Store API - Proxies
export const listStoreProxies = () => {
  return http.get<ProxyListResp>('/api/store/proxies')
}

export const getStoreProxy = (name: string) => {
  return http.get<ProxyDefinition>(
    `/api/store/proxies/${encodeURIComponent(name)}`,
  )
}

export const createStoreProxy = (config: ProxyDefinition) => {
  return http.post<ProxyDefinition>('/api/store/proxies', config)
}

export const updateStoreProxy = (name: string, config: ProxyDefinition) => {
  return http.put<ProxyDefinition>(
    `/api/store/proxies/${encodeURIComponent(name)}`,
    config,
  )
}

export const deleteStoreProxy = (name: string) => {
  return http.delete<void>(`/api/store/proxies/${encodeURIComponent(name)}`)
}

// Store API - Visitors
export const listStoreVisitors = () => {
  return http.get<VisitorListResp>('/api/store/visitors')
}

export const getStoreVisitor = (name: string) => {
  return http.get<VisitorDefinition>(
    `/api/store/visitors/${encodeURIComponent(name)}`,
  )
}

export const createStoreVisitor = (config: VisitorDefinition) => {
  return http.post<VisitorDefinition>('/api/store/visitors', config)
}

export const updateStoreVisitor = (
  name: string,
  config: VisitorDefinition,
) => {
  return http.put<VisitorDefinition>(
    `/api/store/visitors/${encodeURIComponent(name)}`,
    config,
  )
}

export const deleteStoreVisitor = (name: string) => {
  return http.delete<void>(`/api/store/visitors/${encodeURIComponent(name)}`)
}
