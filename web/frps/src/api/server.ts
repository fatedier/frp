import { http } from './http'
import type { ServerInfo } from '../types/server'

export const getServerInfo = () => {
  return http.get<ServerInfo>('../api/serverinfo')
}
