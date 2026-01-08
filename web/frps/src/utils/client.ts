import { formatDistanceToNow } from './format'
import type { ClientInfoData } from '../types/client'

export class Client {
  key: string
  user: string
  clientId: string
  runId: string
  hostname: string
  metas: Map<string, string>
  firstConnectedAt: Date
  lastConnectedAt: Date
  disconnectedAt?: Date
  online: boolean

  constructor(data: ClientInfoData) {
    this.key = data.key
    this.user = data.user
    this.clientId = data.clientId
    this.runId = data.runId
    this.hostname = data.hostname
    this.metas = new Map<string, string>()
    if (data.metas) {
      for (const [key, value] of Object.entries(data.metas)) {
        this.metas.set(key, value)
      }
    }
    this.firstConnectedAt = new Date(data.firstConnectedAt * 1000)
    this.lastConnectedAt = new Date(data.lastConnectedAt * 1000)
    if (data.disconnectedAt && data.disconnectedAt > 0) {
      this.disconnectedAt = new Date(data.disconnectedAt * 1000)
    }
    this.online = data.online
  }

  get displayName(): string {
    if (this.clientId) {
      return this.user ? `${this.user}.${this.clientId}` : this.clientId
    }
    return this.runId
  }

  get shortRunId(): string {
    return this.runId.substring(0, 8)
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

  get statusColor(): string {
    return this.online ? 'success' : 'danger'
  }

  get metasArray(): Array<{ key: string; value: string }> {
    const arr: Array<{ key: string; value: string }> = []
    this.metas.forEach((value, key) => {
      arr.push({ key, value })
    })
    return arr
  }

  matchesFilter(searchText: string): boolean {
    const search = searchText.toLowerCase()
    return (
      this.key.toLowerCase().includes(search) ||
      this.user.toLowerCase().includes(search) ||
      this.clientId.toLowerCase().includes(search) ||
      this.runId.toLowerCase().includes(search) ||
      this.hostname.toLowerCase().includes(search)
    )
  }
}
