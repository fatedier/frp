<template>
  <el-card class="client-card" shadow="hover" :body-style="{ padding: '20px' }">
    <div class="client-header">
      <div class="client-status">
        <span class="status-dot" :class="statusClass"></span>
        <span class="client-name">{{ client.displayName }}</span>
      </div>
      <el-tag :type="client.statusColor" size="small">
        {{ client.online ? 'Online' : 'Offline' }}
      </el-tag>
    </div>

    <div class="client-info">
      <div class="info-row">
        <el-icon class="info-icon"><Monitor /></el-icon>
        <span class="info-label">Hostname:</span>
        <span class="info-value">{{ client.hostname || 'N/A' }}</span>
      </div>

      <div class="info-row" v-if="client.ip">
        <el-icon class="info-icon"><Connection /></el-icon>
        <span class="info-label">IP:</span>
        <span class="info-value monospace">{{ client.ip }}</span>
      </div>

      <div class="info-row" v-if="client.user">
        <el-icon class="info-icon"><User /></el-icon>
        <span class="info-label">User:</span>
        <span class="info-value">{{ client.user }}</span>
      </div>

      <div class="info-row">
        <el-icon class="info-icon"><Key /></el-icon>
        <span class="info-label">Run ID:</span>
        <span class="info-value monospace">{{ client.runID }}</span>
      </div>

      <div class="info-row" v-if="client.firstConnectedAt">
        <el-icon class="info-icon"><Clock /></el-icon>
        <span class="info-label">First Connected:</span>
        <span class="info-value">{{ client.firstConnectedAgo }}</span>
      </div>

      <div class="info-row" v-if="client.online">
        <el-icon class="info-icon"><Clock /></el-icon>
        <span class="info-label">Last Connected:</span>
        <span class="info-value">{{ client.lastConnectedAgo }}</span>
      </div>

      <div class="info-row" v-if="!client.online && client.disconnectedAt">
        <el-icon class="info-icon"><CircleClose /></el-icon>
        <span class="info-label">Disconnected:</span>
        <span class="info-value">{{ client.disconnectedAgo }}</span>
      </div>
    </div>

  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Monitor, User, Key, Clock, CircleClose, Connection } from '@element-plus/icons-vue'
import type { Client } from '../utils/client'

interface Props {
  client: Client
}

const props = defineProps<Props>()

const statusClass = computed(() => {
  return `status-${props.client.statusColor}`
})
</script>

<style scoped>
.client-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 1px solid #e4e7ed;
}

.client-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
}

html.dark .client-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.client-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

html.dark .client-header {
  border-bottom-color: #3a3d5c;
}

.client-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.status-success {
  background-color: #67c23a;
  box-shadow: 0 0 0 0 rgba(103, 194, 58, 0.7);
}

.status-warning {
  background-color: #e6a23c;
  box-shadow: 0 0 0 0 rgba(230, 162, 60, 0.7);
}

.status-danger {
  background-color: #f56c6c;
  box-shadow: 0 0 0 0 rgba(245, 108, 108, 0.7);
}

.client-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

html.dark .client-name {
  color: #e5e7eb;
}

.client-info {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.info-icon {
  color: #909399;
  font-size: 16px;
}

html.dark .info-icon {
  color: #9ca3af;
}

.info-label {
  color: #909399;
  font-weight: 500;
  min-width: 100px;
}

html.dark .info-label {
  color: #9ca3af;
}

.info-value {
  color: #606266;
  flex: 1;
}

html.dark .info-value {
  color: #d1d5db;
}

.monospace {
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
  word-break: break-all;
}
</style>
