import type { ProxyType, VisitorType } from './constants'

export interface ProxyDefinition {
  name: string
  type: ProxyType
  tcp?: Record<string, any>
  udp?: Record<string, any>
  http?: Record<string, any>
  https?: Record<string, any>
  tcpmux?: Record<string, any>
  stcp?: Record<string, any>
  sudp?: Record<string, any>
  xtcp?: Record<string, any>
  xudp?: Record<string, any>
  'xtcp+xudp'?: Record<string, any>
  'tcp+udp'?: Record<string, any>
  'http+https'?: Record<string, any>
  'stcp+sudp'?: Record<string, any>
}

export interface VisitorDefinition {
  name: string
  type: VisitorType
  stcp?: Record<string, any>
  sudp?: Record<string, any>
  xtcp?: Record<string, any>
  xudp?: Record<string, any>
  'xtcp+xudp'?: Record<string, any>
}

export interface ProxyListResp {
  proxies: ProxyDefinition[]
}

export interface VisitorListResp {
  visitors: VisitorDefinition[]
}
