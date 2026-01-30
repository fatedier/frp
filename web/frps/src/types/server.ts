export interface ServerInfo {
  version: string
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

  // Stats
  totalTrafficIn: number
  totalTrafficOut: number
  curConns: number
  clientCounts: number
  proxyTypeCount: Record<string, number>
}
