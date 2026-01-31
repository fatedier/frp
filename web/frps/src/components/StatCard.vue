<template>
  <el-card
    class="stat-card"
    :class="{ clickable: !!to }"
    :body-style="{ padding: '20px' }"
    shadow="hover"
    @click="handleClick"
  >
    <div class="stat-card-content">
      <div class="stat-icon" :class="`icon-${type}`">
        <component :is="iconComponent" class="icon" />
      </div>
      <div class="stat-info">
        <div class="stat-value">{{ value }}</div>
        <div class="stat-label">{{ label }}</div>
      </div>
      <el-icon v-if="to" class="arrow-icon"><ArrowRight /></el-icon>
    </div>
    <div v-if="subtitle" class="stat-subtitle">{{ subtitle }}</div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  User,
  Connection,
  DataAnalysis,
  Promotion,
  ArrowRight,
} from '@element-plus/icons-vue'

interface Props {
  label: string
  value: string | number
  type?: 'clients' | 'proxies' | 'connections' | 'traffic'
  subtitle?: string
  to?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'clients',
})

const router = useRouter()

const iconComponent = computed(() => {
  switch (props.type) {
    case 'clients':
      return User
    case 'proxies':
      return Connection
    case 'connections':
      return DataAnalysis
    case 'traffic':
      return Promotion
    default:
      return User
  }
})

const handleClick = () => {
  if (props.to) {
    router.push(props.to)
  }
}
</script>

<style scoped>
.stat-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 1px solid #e4e7ed;
}

.stat-card.clickable {
  cursor: pointer;
}

.stat-card.clickable:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.1);
}

.stat-card.clickable:hover .arrow-icon {
  transform: translateX(4px);
}

html.dark .stat-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.stat-card-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.arrow-icon {
  color: #909399;
  font-size: 18px;
  transition: transform 0.2s ease;
  flex-shrink: 0;
}

html.dark .arrow-icon {
  color: #9ca3af;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon .icon {
  width: 28px;
  height: 28px;
}

.icon-clients {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.icon-proxies {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.icon-connections {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
}

.icon-traffic {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: white;
}

html.dark .icon-clients {
  background: linear-gradient(135deg, #818cf8 0%, #a78bfa 100%);
}

html.dark .icon-proxies {
  background: linear-gradient(135deg, #fb7185 0%, #f43f5e 100%);
}

html.dark .icon-connections {
  background: linear-gradient(135deg, #60a5fa 0%, #3b82f6 100%);
}

html.dark .icon-traffic {
  background: linear-gradient(135deg, #34d399 0%, #10b981 100%);
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 28px;
  font-weight: 500;
  line-height: 1.2;
  color: #303133;
  margin-bottom: 4px;
}

html.dark .stat-value {
  color: #e5e7eb;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  font-weight: 500;
}

html.dark .stat-label {
  color: #9ca3af;
}

.stat-subtitle {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e4e7ed;
  font-size: 12px;
  color: #909399;
}

html.dark .stat-subtitle {
  border-top-color: #3a3d5c;
  color: #9ca3af;
}
</style>
