<template>
  <router-link :to="proxyLink" class="proxy-card">
    <div class="card-main">
      <div class="card-left">
        <div class="card-header">
          <span class="proxy-name">{{ proxy.name }}</span>
          <span v-if="showType" class="type-tag">{{
            proxy.type.toUpperCase()
          }}</span>
        </div>

        <div class="card-meta">
          <span v-if="proxy.port" class="meta-item">
            <span class="meta-label">Port:</span>
            <span class="meta-value">{{ proxy.port }}</span>
          </span>
          <span class="meta-item">
            <span class="meta-label">Connections:</span>
            <span class="meta-value">{{ proxy.conns }}</span>
          </span>
          <span class="meta-item" v-if="proxy.clientID">
            <span class="meta-label">Client:</span>
            <span class="meta-value">{{
              proxy.user ? `${proxy.user}.${proxy.clientID}` : proxy.clientID
            }}</span>
          </span>
        </div>
      </div>

      <div class="card-right">
        <div class="traffic-stats">
          <div class="traffic-row">
            <el-icon class="traffic-icon out"><Top /></el-icon>
            <span class="traffic-value">{{
              formatFileSize(proxy.trafficOut)
            }}</span>
          </div>
          <div class="traffic-row">
            <el-icon class="traffic-icon in"><Bottom /></el-icon>
            <span class="traffic-value">{{
              formatFileSize(proxy.trafficIn)
            }}</span>
          </div>
        </div>

        <div class="status-badge" :class="proxy.status">
          {{ proxy.status }}
        </div>
      </div>
    </div>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Top, Bottom } from '@element-plus/icons-vue'
import { formatFileSize } from '../utils/format'
import type { BaseProxy } from '../utils/proxy'

interface Props {
  proxy: BaseProxy
  showType?: boolean
}

const props = defineProps<Props>()
const route = useRoute()

const proxyLink = computed(() => {
  const base = `/proxy/${props.proxy.name}`
  // If we're on a client detail page, pass client info
  if (route.name === 'ClientDetail' && route.params.key) {
    return `${base}?from=client&client=${route.params.key}`
  }
  return base
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
  text-decoration: none;
  cursor: pointer;
}

.proxy-card:hover {
  border-color: var(--el-border-color-light);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
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
  gap: 8px;
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
  gap: 24px;
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

/* Right Section */
.card-right {
  display: flex;
  align-items: center;
  gap: 24px;
  flex-shrink: 0;
}

.traffic-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}

.traffic-row {
  display: flex;
  align-items: center;
  gap: 6px;
  line-height: 1;
}

.traffic-icon {
  font-size: 12px;
}

.traffic-icon.in {
  color: var(--el-color-primary);
}

.traffic-icon.out {
  color: var(--el-color-success);
}

.traffic-value {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  text-align: right;
}

.status-badge {
  display: inline-flex;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
  text-transform: capitalize;
}

.status-badge.online {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.status-badge.offline {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
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

  .traffic-stats {
    align-items: flex-start;
  }
}
</style>
