import { http } from './http'
import type { ClientInfoData } from '../types/client'

export const getClients = () => {
  return http.get<ClientInfoData[]>('../api/clients')
}

export const getClient = (key: string) => {
  return http.get<ClientInfoData>(`../api/clients/${key}`)
}
