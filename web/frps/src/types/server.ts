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
  transportProtocol: string
  autoTransportEnabled: boolean
  autoTransportProtocols?: string[]

  // Stats
  totalTrafficIn: number
  totalTrafficOut: number
  curConns: number
  clientCounts: number
  proxyTypeCount: Record<string, number>
  autoNegotiationSuccess?: number
  autoNegotiationFailure?: number
  autoTransportSelections?: Record<string, number>
  autoTransportClientCounts?: Record<string, number>
  autoTransportSwitchCounts?: Record<string, number>
  autoTransportIllegalSelections?: Record<string, number>
}
