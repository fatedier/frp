export interface ServerInfo {
  version: string
  config: ServerInfoConfig
  status: ServerInfoStatus
}

export interface ServerInfoConfig {
  bindPort: number
  vhostHTTPPort: number
  vhostHTTPSPort: number
  tcpmuxHTTPConnectPort: number
  kcpBindPort: number
  quicBindPort: number
  subdomainHost: string
  maxPoolCount: number
  maxPortsPerClient: number
  heartbeatTimeout: number
  allowPortsStr: string
  tlsForce: boolean
}

export interface ServerInfoStatus {
  totalTrafficIn: number
  totalTrafficOut: number
  curConns: number
  clientCounts: number
  proxyTypeCount: Record<string, number>
}
