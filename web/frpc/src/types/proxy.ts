export interface ProxyStatus {
  name: string
  type: string
  status: string
  err: string
  local_addr: string
  plugin: string
  remote_addr: string
  [key: string]: any
}

export type StatusResponse = Record<string, ProxyStatus[]>
