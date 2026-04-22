export interface AutoTransportScoreDetail {
  strategy?: string
  total: number
  successes: number
  probeCount: number
  successRate: number
  avgRTTMs: number
  priority: number
  successScore: number
  latencyPenalty: number
  priorityPenalty: number
  lastGoodBonus?: number
  failurePenalty?: number
}

export interface TransportStatus {
  autoEnabled: boolean
  state: string
  currentProtocol?: string
  currentAddr?: string
  currentPort?: number
  currentScore?: number
  previousProtocol?: string
  lastGoodProtocol?: string
  lastSwitchReason?: string
  lastError?: string
  switchCount: number
  dynamic: boolean
  stickyRemainingSec?: number
  cooldownRemainingSec?: number
  blacklistProtocols?: string[]
  strategy?: string
  lastScores?: Record<string, number>
  lastScoreDetails?: Record<string, AutoTransportScoreDetail>
  lastSuccessRates?: Record<string, number>
  lastProbeRTTMs?: Record<string, number>
  lastProbeErrors?: Record<string, string>
  heartbeatRTTMs?: number
  avgHeartbeatRTTMs?: number
  heartbeatTimeouts?: number
  workConnFailures?: number
  qualityDegradeCount?: number
  degradeEvents?: number
  persistLastGood: boolean
}
