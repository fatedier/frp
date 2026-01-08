export interface ClientInfoData {
  key: string
  user: string
  clientId: string
  runId: string
  hostname: string
  metas?: Record<string, string>
  firstConnectedAt: number
  lastConnectedAt: number
  disconnectedAt?: number
  online: boolean
}
