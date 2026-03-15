export const PROXY_TYPES = [
  'tcp',
  'udp',
  'http',
  'https',
  'tcpmux',
  'stcp',
  'sudp',
  'xtcp',
] as const

export type ProxyType = (typeof PROXY_TYPES)[number]

export const VISITOR_TYPES = ['stcp', 'sudp', 'xtcp'] as const

export type VisitorType = (typeof VISITOR_TYPES)[number]

export const PLUGIN_TYPES = [
  '',
  'http2https',
  'http_proxy',
  'https2http',
  'https2https',
  'http2http',
  'socks5',
  'static_file',
  'unix_domain_socket',
  'tls2raw',
  'virtual_net',
] as const

export type PluginType = (typeof PLUGIN_TYPES)[number]
