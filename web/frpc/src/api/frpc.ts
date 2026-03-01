import { http } from './http'
import type {
  StatusResponse,
  StoreProxyListResp,
  StoreProxyConfig,
  StoreVisitorListResp,
  StoreVisitorConfig,
} from '../types/proxy'

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

// Store API - Proxies
export const listStoreProxies = () => {
  return http.get<StoreProxyListResp>('/api/store/proxies')
}

export const getStoreProxy = (name: string) => {
  return http.get<StoreProxyConfig>(
    `/api/store/proxies/${encodeURIComponent(name)}`,
  )
}

export const createStoreProxy = (config: Record<string, any>) => {
  return http.post<void>('/api/store/proxies', config)
}

export const updateStoreProxy = (name: string, config: Record<string, any>) => {
  return http.put<void>(
    `/api/store/proxies/${encodeURIComponent(name)}`,
    config,
  )
}

export const deleteStoreProxy = (name: string) => {
  return http.delete<void>(`/api/store/proxies/${encodeURIComponent(name)}`)
}

// Store API - Visitors
export const listStoreVisitors = () => {
  return http.get<StoreVisitorListResp>('/api/store/visitors')
}

export const getStoreVisitor = (name: string) => {
  return http.get<StoreVisitorConfig>(
    `/api/store/visitors/${encodeURIComponent(name)}`,
  )
}

export const createStoreVisitor = (config: Record<string, any>) => {
  return http.post<void>('/api/store/visitors', config)
}

export const updateStoreVisitor = (
  name: string,
  config: Record<string, any>,
) => {
  return http.put<void>(
    `/api/store/visitors/${encodeURIComponent(name)}`,
    config,
  )
}

export const deleteStoreVisitor = (name: string) => {
  return http.delete<void>(`/api/store/visitors/${encodeURIComponent(name)}`)
}
