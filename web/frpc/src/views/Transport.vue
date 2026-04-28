<template>
  <div class="transport-page" v-loading="loading">
    <div class="page-top">
      <div class="page-header">
        <h2 class="page-title">Transport</h2>
        <ActionButton variant="outline" size="small" @click="fetchData">
          <el-icon><Refresh /></el-icon>
        </ActionButton>
      </div>

      <div class="summary-grid">
        <div class="summary-item">
          <span class="summary-label">Protocol</span>
          <span class="summary-value">{{ currentEndpoint }}</span>
        </div>
        <div class="summary-item">
          <span class="summary-label">State</span>
          <span class="status-pill" :class="stateClass">
            <span class="status-dot"></span>
            {{ status?.state || 'UNKNOWN' }}
          </span>
        </div>
        <div class="summary-item">
          <span class="summary-label">Strategy</span>
          <span class="summary-value">{{ status?.strategy || 'static' }}</span>
        </div>
        <div class="summary-item">
          <span class="summary-label">Switches</span>
          <span class="summary-value">{{ status?.switchCount ?? 0 }}</span>
        </div>
      </div>
    </div>

    <div class="page-content">
      <section class="section">
        <div class="section-header">
          <h3>Selection</h3>
        </div>
        <div class="detail-grid">
          <div class="detail-item">
            <span>Current Score</span>
            <strong>{{ status?.currentScore ?? 0 }}</strong>
          </div>
          <div class="detail-item">
            <span>Previous</span>
            <strong>{{ status?.previousProtocol || '-' }}</strong>
          </div>
          <div class="detail-item">
            <span>Last Good</span>
            <strong>{{ status?.lastGoodProtocol || '-' }}</strong>
          </div>
          <div class="detail-item">
            <span>Dynamic</span>
            <strong>{{ status?.dynamic ? 'yes' : 'no' }}</strong>
          </div>
          <div class="detail-item">
            <span>Sticky</span>
            <strong>{{ formatSeconds(status?.stickyRemainingSec) }}</strong>
          </div>
          <div class="detail-item">
            <span>Cooldown</span>
            <strong>{{ formatSeconds(status?.cooldownRemainingSec) }}</strong>
          </div>
        </div>
      </section>

      <section class="section">
        <div class="section-header">
          <h3>Candidate Scores</h3>
        </div>
        <el-table v-if="scoreRows.length > 0" :data="scoreRows" size="small">
          <el-table-column prop="endpoint" label="Endpoint" min-width="190" />
          <el-table-column prop="score" label="Score" width="96" align="right" />
          <el-table-column label="Success" width="96" align="right">
            <template #default="{ row }">{{ formatPercent(row.successRate) }}</template>
          </el-table-column>
          <el-table-column prop="avgRTTMs" label="RTT ms" width="88" align="right" />
          <el-table-column prop="successScore" label="Success Score" width="122" align="right" />
          <el-table-column prop="latencyPenalty" label="RTT Penalty" width="112" align="right" />
          <el-table-column prop="priorityPenalty" label="Priority Penalty" width="132" align="right" />
          <el-table-column prop="failurePenalty" label="Failure Penalty" width="128" align="right" />
          <el-table-column prop="error" label="Error" min-width="160" />
        </el-table>
        <div v-else class="empty-state">No candidate scores</div>
      </section>

      <section class="section">
        <div class="section-header">
          <h3>Runtime</h3>
        </div>
        <div class="detail-grid">
          <div class="detail-item">
            <span>Heartbeat RTT</span>
            <strong>{{ status?.heartbeatRTTMs ?? 0 }} ms</strong>
          </div>
          <div class="detail-item">
            <span>Average RTT</span>
            <strong>{{ status?.avgHeartbeatRTTMs ?? 0 }} ms</strong>
          </div>
          <div class="detail-item">
            <span>Heartbeat Timeouts</span>
            <strong>{{ status?.heartbeatTimeouts ?? 0 }}</strong>
          </div>
          <div class="detail-item">
            <span>Work Conn Failures</span>
            <strong>{{ status?.workConnFailures ?? 0 }}</strong>
          </div>
          <div class="detail-item">
            <span>Quality Degrade</span>
            <strong>{{ status?.qualityDegradeCount ?? 0 }}</strong>
          </div>
          <div class="detail-item">
            <span>Degrade Events</span>
            <strong>{{ status?.degradeEvents ?? 0 }}</strong>
          </div>
        </div>
      </section>

      <section v-if="hasEvents" class="section">
        <div class="section-header">
          <h3>Events</h3>
        </div>
        <div class="event-list">
          <div v-if="status?.lastSwitchReason" class="event-item">
            <span>Reason</span>
            <strong>{{ status.lastSwitchReason }}</strong>
          </div>
          <div v-if="status?.lastError" class="event-item error">
            <span>Error</span>
            <strong>{{ status.lastError }}</strong>
          </div>
          <div v-if="blacklistText" class="event-item">
            <span>Blacklist</span>
            <strong>{{ blacklistText }}</strong>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import { getTransportStatus } from '../api/frpc'
import type { AutoTransportScoreDetail, TransportStatus } from '../types'

type ScoreRow = AutoTransportScoreDetail & {
  endpoint: string
  score: number
  error: string
}

const status = ref<TransportStatus | null>(null)
const loading = ref(false)

const currentEndpoint = computed(() => {
  if (!status.value?.currentProtocol) return '-'
  const port = status.value.currentPort ? `:${status.value.currentPort}` : ''
  return `${status.value.currentProtocol}@${status.value.currentAddr || ''}${port}`
})

const stateClass = computed(() => {
  switch (status.value?.state) {
    case 'CONNECTED':
    case 'SELECTED':
      return 'running'
    case 'DEGRADED':
    case 'BACKOFF':
    case 'SWITCHING':
      return 'waiting'
    case 'FALLBACK_STATIC':
    case 'STATIC':
      return 'disabled'
    default:
      return status.value?.lastError ? 'error' : 'waiting'
  }
})

const scoreRows = computed<ScoreRow[]>(() => {
  if (!status.value) return []
  const details = status.value.lastScoreDetails || {}
  const scores = status.value.lastScores || {}
  const rates = status.value.lastSuccessRates || {}
  const rtts = status.value.lastProbeRTTMs || {}
  const errors = status.value.lastProbeErrors || {}
  const endpoints = new Set([
    ...Object.keys(details),
    ...Object.keys(scores),
    ...Object.keys(errors),
  ])

  return Array.from(endpoints)
    .sort()
    .map((endpoint) => {
      const detail = details[endpoint]
      return {
        endpoint,
        strategy: detail?.strategy || status.value?.strategy,
        total: detail?.total ?? scores[endpoint] ?? 0,
        score: detail?.total ?? scores[endpoint] ?? 0,
        successes: detail?.successes ?? 0,
        probeCount: detail?.probeCount ?? 0,
        successRate: detail?.successRate ?? rates[endpoint] ?? 0,
        avgRTTMs: detail?.avgRTTMs ?? rtts[endpoint] ?? 0,
        priority: detail?.priority ?? 0,
        successScore: detail?.successScore ?? 0,
        latencyPenalty: detail?.latencyPenalty ?? 0,
        priorityPenalty: detail?.priorityPenalty ?? 0,
        lastGoodBonus: detail?.lastGoodBonus ?? 0,
        failurePenalty: detail?.failurePenalty ?? 0,
        error: errors[endpoint] || '',
      }
    })
})

const blacklistText = computed(() => {
  const values = status.value?.blacklistProtocols || []
  return values.length > 0 ? values.join(', ') : ''
})

const hasEvents = computed(() => {
  return !!status.value?.lastSwitchReason || !!status.value?.lastError || !!blacklistText.value
})

const formatPercent = (value: number) => `${Math.round(value * 100)}%`

const formatSeconds = (value?: number) => {
  if (!value) return '0s'
  return `${value}s`
}

const fetchData = async () => {
  loading.value = true
  try {
    status.value = await getTransportStatus()
  } catch (err: any) {
    ElMessage.error('Failed to get transport status: ' + err.message)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped lang="scss">
.transport-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1120px;
  margin: 0 auto;
}

.page-top {
  flex-shrink: 0;
  padding: $spacing-xl 40px 0;
}

.page-content {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-xl 40px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-xl;
}

.summary-grid,
.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: $spacing-md;
}

.summary-item,
.detail-item,
.event-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: $spacing-md;
  border: 1px solid $color-border-light;
  border-radius: $radius-md;
  background: $color-bg-tertiary;
  min-width: 0;
}

.summary-label,
.detail-item span,
.event-item span {
  font-size: $font-size-xs;
  color: $color-text-muted;
  font-weight: $font-weight-medium;
}

.summary-value,
.detail-item strong,
.event-item strong {
  color: $color-text-primary;
  font-size: $font-size-md;
  font-weight: $font-weight-semibold;
  overflow-wrap: anywhere;
}

.section {
  margin-bottom: $spacing-xl;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: $spacing-md;

  h3 {
    margin: 0;
    color: $color-text-primary;
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
  }
}

.empty-state {
  padding: $spacing-xl;
  text-align: center;
  border: 1px solid $color-border-light;
  border-radius: $radius-md;
  color: $color-text-muted;
  background: $color-bg-tertiary;
}

.event-list {
  display: grid;
  gap: $spacing-md;
}

.event-item.error strong {
  color: $color-danger;
}

:deep(.el-table) {
  border-radius: $radius-md;
  border: 1px solid $color-border-light;
  overflow: hidden;
}

@include mobile {
  .page-top {
    padding: $spacing-lg $spacing-lg 0;
  }

  .page-content {
    padding: $spacing-lg;
  }

  .summary-grid,
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
