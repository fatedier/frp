export interface ClientInfoData {
  key: string
  user: string
  clientID: string
  runID: string
  hostname: string
  clientIP?: string
  metas?: Record<string, string>
  firstConnectedAt: number
  lastConnectedAt: number
  disconnectedAt?: number
  online: boolean
}
