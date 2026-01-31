import { http } from './http'
import type { StatusResponse } from '../types/proxy'

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
