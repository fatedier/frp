<template>
  <div class="client-card" @click="viewDetail">
    <div class="card-icon-wrapper">
      <div
        class="status-dot-large"
        :class="client.online ? 'online' : 'offline'"
      ></div>
    </div>

    <div class="card-content">
      <div class="card-header">
        <span class="client-main-id">{{ client.displayName }}</span>
        <span v-if="client.hostname" class="hostname-badge">{{
          client.hostname
        }}</span>
      </div>

      <div class="card-meta">
        <div class="meta-group">
          <span v-if="client.ip" class="meta-item">
            <span class="meta-label">IP</span>
            <span class="meta-value">{{ client.ip }}</span>
          </span>
        </div>
        <span class="meta-item activity">
          <el-icon class="activity-icon"><DataLine /></el-icon>
          <span class="meta-value">{{
            client.online ? client.lastConnectedAgo : client.disconnectedAgo
          }}</span>
        </span>
      </div>
    </div>

    <div class="card-action">
      <div class="status-badge" :class="client.online ? 'online' : 'offline'">
        {{ client.online ? 'Online' : 'Offline' }}
      </div>
      <el-icon class="arrow-icon"><ArrowRight /></el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { DataLine, ArrowRight } from '@element-plus/icons-vue'
import type { Client } from '../utils/client'

interface Props {
  client: Client
}

const props = defineProps<Props>()
const router = useRouter()

const viewDetail = () => {
  router.push({
    name: 'ClientDetail',
    params: { key: props.client.key },
  })
}
</script>

<style scoped>
.client-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 24px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.client-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.04);
  border-color: var(--el-border-color-light);
}

.card-icon-wrapper {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--el-fill-color);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s;
}

.client-card:hover .card-icon-wrapper {
  background: var(--el-color-success-light-9);
}

.status-dot-large {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  transition: all 0.3s;
}

.status-dot-large.online {
  background-color: var(--el-color-success);
  box-shadow: 0 0 0 2px var(--el-color-success-light-8);
}

.status-dot-large.offline {
  background-color: var(--el-text-color-placeholder);
}

.card-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.client-main-id {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.hostname-badge {
  font-size: 12px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--el-fill-color-dark);
  color: var(--el-text-color-regular);
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 24px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  flex-wrap: wrap;
}

.meta-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-label {
  color: var(--el-text-color-placeholder);
  font-weight: 500;
  font-size: 13px;
}

.meta-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.activity .meta-value {
  font-weight: 400;
  color: var(--el-text-color-secondary);
}

.card-action {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-shrink: 0;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.status-badge.online {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
}

.status-badge.offline {
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}

.arrow-icon {
  font-size: 18px;
  color: var(--el-text-color-placeholder);
  transition: all 0.2s;
}

.client-card:hover .arrow-icon {
  color: var(--el-text-color-primary);
  transform: translateX(4px);
}

/* Dark mode adjustments */
html.dark .card-icon-wrapper {
  background: var(--el-fill-color-light);
}

html.dark .client-card:hover .card-icon-wrapper {
  background: var(--el-color-success-light-9);
}

html.dark .status-dot-large.online {
  box-shadow: 0 0 0 2px rgba(var(--el-color-success-rgb), 0.2);
}

@media (max-width: 640px) {
  .client-card {
    flex-direction: column;
    align-items: flex-start;
    padding: 20px;
  }

  .card-icon-wrapper {
    width: 48px;
    height: 48px;
  }

  .card-content {
    width: 100%;
    gap: 12px;
  }

  .card-action {
    width: 100%;
    justify-content: space-between;
    padding-top: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}
</style>
