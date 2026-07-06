import { http } from './http'
import type { ServerInfo } from '../types/server'

export const getServerInfo = () => {
  return http.getV2<ServerInfo>('../api/v2/system/info')
}
