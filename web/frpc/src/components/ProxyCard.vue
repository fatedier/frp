<template>
  <div
    class="proxy-card"
    :class="{ 'has-error': proxy.err, 'is-store': isStore }"
  >
    <div class="card-main">
      <div class="card-left">
        <div class="card-header">
          <span class="proxy-name">{{ proxy.name }}</span>
          <span class="type-tag" :class="`type-${proxy.type}`">{{
            proxy.type.toUpperCase()
          }}</span>
          <span v-if="isStore" class="source-tag">
            <svg
              class="store-icon"
              viewBox="0 0 16 16"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M2 4.5A1.5 1.5 0 013.5 3h9A1.5 1.5 0 0114 4.5v1a.5.5 0 01-.5.5h-11a.5.5 0 01-.5-.5v-1z"
                fill="currentColor"
              />
              <path
                d="M3 7v5.5A1.5 1.5 0 004.5 14h7a1.5 1.5 0 001.5-1.5V7H3zm4 2h2a.5.5 0 010 1H7a.5.5 0 010-1z"
                fill="currentColor"
              />
            </svg>
            Store
          </span>
        </div>

        <div class="card-meta">
          <span v-if="proxy.local_addr" class="meta-item">
            <span class="meta-label">Local</span>
            <span class="meta-value code">{{ proxy.local_addr }}</span>
          </span>
          <span v-if="proxy.plugin" class="meta-item">
            <span class="meta-label">Plugin</span>
            <span class="meta-value code">{{ proxy.plugin }}</span>
          </span>
          <span v-if="proxy.remote_addr" class="meta-item">
            <span class="meta-label">Remote</span>
            <span class="meta-value code">{{ proxy.remote_addr }}</span>
          </span>
        </div>
      </div>

      <div class="card-right">
        <div v-if="proxy.err" class="error-info">
          <el-tooltip :content="proxy.err" placement="top" :show-after="300">
            <div class="error-badge">
              <el-icon class="error-icon"><Warning /></el-icon>
              <span class="error-text">Error</span>
            </div>
          </el-tooltip>
        </div>

        <div class="status-badge" :class="statusClass">
          <span class="status-dot"></span>
          {{ proxy.status }}
        </div>

        <!-- Store actions -->
        <div v-if="isStore" class="card-actions">
          <button
            class="action-btn edit-btn"
            @click.stop="$emit('edit', proxy)"
          >
            <svg
              viewBox="0 0 16 16"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M11.293 1.293a1 1 0 011.414 0l2 2a1 1 0 010 1.414l-9 9A1 1 0 015 14H3a1 1 0 01-1-1v-2a1 1 0 01.293-.707l9-9z"
                fill="currentColor"
              />
            </svg>
          </button>
          <button
            class="action-btn delete-btn"
            @click.stop="$emit('delete', proxy)"
          >
            <svg
              viewBox="0 0 16 16"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M5.5 5.5A.5.5 0 016 6v6a.5.5 0 01-1 0V6a.5.5 0 01.5-.5zm2.5 0a.5.5 0 01.5.5v6a.5.5 0 01-1 0V6a.5.5 0 01.5-.5zm3 .5a.5.5 0 00-1 0v6a.5.5 0 001 0V6z"
                fill="currentColor"
              />
              <path
                fill-rule="evenodd"
                clip-rule="evenodd"
                d="M14.5 3a1 1 0 01-1 1H13v9a2 2 0 01-2 2H5a2 2 0 01-2-2V4h-.5a1 1 0 010-2H6a1 1 0 011-1h2a1 1 0 011 1h3.5a1 1 0 011 1zM4.118 4L4 4.059V13a1 1 0 001 1h6a1 1 0 001-1V4.059L11.882 4H4.118zM6 2h4v1H6V2z"
                fill="currentColor"
              />
            </svg>
          </button>
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

defineEmits<{
  edit: [proxy: ProxyStatus]
  delete: [proxy: ProxyStatus]
}>()

const isStore = computed(() => props.proxy.source === 'store')

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
  position: relative;
  display: block;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.proxy-card:hover {
  border-color: var(--el-border-color);
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.06),
    0 1px 4px rgba(0, 0, 0, 0.04);
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
  padding: 18px 20px;
  gap: 20px;
  min-height: 76px;
}

/* Left Section */
.card-left {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.proxy-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.3;
  letter-spacing: -0.01em;
}

.type-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.type-tag.type-tcp {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}
.type-tag.type-udp {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}
.type-tag.type-http {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}
.type-tag.type-https {
  background: rgba(16, 185, 129, 0.15);
  color: #059669;
}
.type-tag.type-stcp,
.type-tag.type-sudp,
.type-tag.type-xtcp {
  background: rgba(139, 92, 246, 0.1);
  color: #8b5cf6;
}
.type-tag.type-tcpmux {
  background: rgba(236, 72, 153, 0.1);
  color: #ec4899;
}

html.dark .type-tag.type-tcp {
  background: rgba(96, 165, 250, 0.15);
  color: #60a5fa;
}
html.dark .type-tag.type-udp {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
}
html.dark .type-tag.type-http {
  background: rgba(52, 211, 153, 0.15);
  color: #34d399;
}
html.dark .type-tag.type-https {
  background: rgba(52, 211, 153, 0.2);
  color: #34d399;
}
html.dark .type-tag.type-stcp,
html.dark .type-tag.type-sudp,
html.dark .type-tag.type-xtcp {
  background: rgba(167, 139, 250, 0.15);
  color: #a78bfa;
}
html.dark .type-tag.type-tcpmux {
  background: rgba(244, 114, 182, 0.15);
  color: #f472b6;
}

.source-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 500;
  padding: 2px 6px;
  border-radius: 4px;
  background: linear-gradient(
    135deg,
    rgba(102, 126, 234, 0.1) 0%,
    rgba(118, 75, 162, 0.1) 100%
  );
  color: #764ba2;
}

html.dark .source-tag {
  background: linear-gradient(
    135deg,
    rgba(129, 140, 248, 0.15) 0%,
    rgba(167, 139, 250, 0.15) 100%
  );
  color: #a78bfa;
}

.store-icon {
  width: 12px;
  height: 12px;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  line-height: 1;
}

.meta-label {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  font-weight: 500;
}

.meta-value {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.meta-value.code {
  font-family:
    'SF Mono', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  background: var(--el-fill-color-light);
  padding: 3px 7px;
  border-radius: 5px;
  font-size: 11px;
  letter-spacing: -0.02em;
}

/* Right Section */
.card-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.error-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  background: var(--el-color-danger-light-9);
  cursor: help;
}

.error-icon {
  color: var(--el-color-danger);
  font-size: 14px;
}

.error-text {
  font-size: 11px;
  font-weight: 500;
  color: var(--el-color-danger);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-badge.running {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}
.status-badge.running .status-dot {
  background: var(--el-color-success);
  box-shadow: 0 0 0 2px var(--el-color-success-light-7);
  animation: pulse 2s infinite;
}

.status-badge.error {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}
.status-badge.error .status-dot {
  background: var(--el-color-danger);
}

.status-badge.waiting {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}
.status-badge.waiting .status-dot {
  background: var(--el-color-warning);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Action buttons */
.card-actions {
  display: none;
  gap: 4px;
}

.proxy-card.is-store:hover .status-badge {
  display: none;
}

.proxy-card:hover .card-actions {
  display: flex;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn svg {
  width: 14px;
  height: 14px;
}

.action-btn:hover {
  transform: scale(1.05);
}

.edit-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.delete-btn:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

html.dark .edit-btn:hover {
  background: rgba(96, 165, 250, 0.15);
  color: #60a5fa;
}

html.dark .delete-btn:hover {
  background: rgba(248, 113, 113, 0.15);
  color: #f87171;
}

/* Mobile Responsive */
@media (max-width: 768px) {
  .card-main {
    flex-direction: column;
    align-items: stretch;
    gap: 14px;
    padding: 14px 16px;
  }

  .card-right {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 14px;
  }

  .card-actions {
    opacity: 1;
    transform: none;
  }
}
</style>
