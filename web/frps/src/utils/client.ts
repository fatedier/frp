import { formatDistanceToNow } from './format'
import type { ClientInfoData, ClientStatus } from '../types/client'

export class Client {
  key: string
  user: string
  clientID: string
  runID: string
  version: string
  wireProtocol: string
  hostname: string
  ip: string
  firstConnectedAt: Date
  lastConnectedAt: Date
  disconnectedAt?: Date
  online: boolean
  status: ClientStatus

  constructor(data: ClientInfoData) {
    this.key = data.key
    this.user = data.user
    this.clientID = data.clientID
    this.runID = data.runID
    this.version = data.version || ''
    this.wireProtocol = data.wireProtocol || ''
    this.hostname = data.hostname
    this.ip = data.clientIP || ''
    this.firstConnectedAt = new Date(data.firstConnectedAt * 1000)
    this.lastConnectedAt = new Date(data.lastConnectedAt * 1000)
    if (data.disconnectedAt && data.disconnectedAt > 0) {
      this.disconnectedAt = new Date(data.disconnectedAt * 1000)
    }
    this.online = data.online
    this.status = data.status || {
      phase: this.online ? 'online' : 'offline',
      curConns: 0,
      proxyCount: 0,
    }
  }

  get displayName(): string {
    if (this.clientID) {
      return this.user ? `${this.user}.${this.clientID}` : this.clientID
    }
    return this.runID
  }

  get wireProtocolLabel(): string {
    if (!this.wireProtocol) return ''
    return `Protocol ${this.wireProtocol}`
  }

  get firstConnectedAgo(): string {
    return formatDistanceToNow(this.firstConnectedAt)
  }

  get lastConnectedAgo(): string {
    return formatDistanceToNow(this.lastConnectedAt)
  }

  get disconnectedAgo(): string {
    if (!this.disconnectedAt) return ''
    return formatDistanceToNow(this.disconnectedAt)
  }
}
