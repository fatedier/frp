<template>
  <div class="proxy-card" :class="{ 'has-error': proxy.err }">
    <div class="card-main">
      <div class="card-left">
        <div class="card-header">
          <span class="proxy-name">{{ proxy.name }}</span>
          <span class="type-tag">{{ proxy.type.toUpperCase() }}</span>
        </div>

        <div class="card-meta">
          <span v-if="proxy.local_addr" class="meta-item">
            <span class="meta-label">Local:</span>
            <span class="meta-value code">{{ proxy.local_addr }}</span>
          </span>
          <span v-if="proxy.plugin" class="meta-item">
            <span class="meta-label">Plugin:</span>
            <span class="meta-value code">{{ proxy.plugin }}</span>
          </span>
          <span v-if="proxy.remote_addr" class="meta-item">
            <span class="meta-label">Remote:</span>
            <span class="meta-value code">{{ proxy.remote_addr }}</span>
          </span>
        </div>
      </div>

      <div class="card-right">
        <div v-if="proxy.err" class="error-info">
          <el-icon class="error-icon"><Warning /></el-icon>
          <span class="error-text">{{ proxy.err }}</span>
        </div>
        <div class="status-badge" :class="statusClass">
          {{ proxy.status }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Warning } from '@element-plus/icons-vue'
import type { ProxyStatus } from '../types/proxy'

interface Props {
  proxy: ProxyStatus
}

const props = defineProps<Props>()

const statusClass = computed(() => {
  switch (props.proxy.status) {
    case 'running':
      return 'running'
    case 'error':
      return 'error'
    default:
      return 'waiting'
  }
})
</script>

<style scoped>
.proxy-card {
  display: block;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  transition: all 0.2s ease-in-out;
  overflow: hidden;
}

.proxy-card:hover {
  border-color: var(--el-border-color-light);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
}

.proxy-card.has-error {
  border-color: var(--el-color-danger-light-5);
}

html.dark .proxy-card.has-error {
  border-color: var(--el-color-danger-dark-2);
}

.card-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  gap: 24px;
  min-height: 80px;
}

/* Left Section */
.card-left {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.proxy-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.type-tag {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  line-height: 1;
}

.meta-label {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
  font-weight: 500;
}

.meta-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.meta-value.code {
  font-family:
    ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

/* Right Section */
.card-right {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}

.error-info {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 200px;
}

.error-icon {
  color: var(--el-color-danger);
  font-size: 16px;
  flex-shrink: 0;
}

.error-text {
  font-size: 12px;
  color: var(--el-color-danger);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-badge {
  display: inline-flex;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.status-badge.running {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.status-badge.error {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}

.status-badge.waiting {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .card-main {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
    padding: 16px;
  }

  .card-right {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 16px;
  }

  .error-info {
    max-width: none;
    flex: 1;
  }
}
</style>
