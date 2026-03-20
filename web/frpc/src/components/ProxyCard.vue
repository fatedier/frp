<template>
  <div class="proxy-card" :class="{ 'has-error': proxy.err }" @click="$emit('click', proxy)">
    <div class="card-main">
      <div class="card-left">
        <div class="card-header">
          <span class="proxy-name">{{ proxy.name }}</span>
          <span class="type-tag">{{ proxy.type.toUpperCase() }}</span>
          <span class="status-pill" :class="statusClass">
            <span class="status-dot"></span>
            {{ proxy.status }}
          </span>
        </div>
        <div class="card-address">
          <template v-if="proxy.remote_addr && localDisplay">
            {{ proxy.remote_addr }} → {{ localDisplay }}
          </template>
          <template v-else-if="proxy.remote_addr">{{ proxy.remote_addr }}</template>
          <template v-else-if="localDisplay">{{ localDisplay }}</template>
        </div>
      </div>
      <div class="card-right">
        <span v-if="showSource" class="source-label">{{ displaySource }}</span>
        <div v-if="showActions" @click.stop>
          <PopoverMenu :width="120" placement="bottom-end">
            <template #trigger>
              <ActionButton variant="outline" size="small">
                <el-icon><MoreFilled /></el-icon>
              </ActionButton>
            </template>
            <PopoverMenuItem v-if="proxy.status === 'disabled'" @click="$emit('toggle', proxy, true)">
              <el-icon><Open /></el-icon>
              Enable
            </PopoverMenuItem>
            <PopoverMenuItem v-else @click="$emit('toggle', proxy, false)">
              <el-icon><TurnOff /></el-icon>
              Disable
            </PopoverMenuItem>
            <PopoverMenuItem @click="$emit('edit', proxy)">
              <el-icon><Edit /></el-icon>
              Edit
            </PopoverMenuItem>
            <PopoverMenuItem danger @click="$emit('delete', proxy)">
              <el-icon><Delete /></el-icon>
              Delete
            </PopoverMenuItem>
          </PopoverMenu>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { MoreFilled, Edit, Delete, Open, TurnOff } from '@element-plus/icons-vue'
import ActionButton from '@shared/components/ActionButton.vue'
import PopoverMenu from '@shared/components/PopoverMenu.vue'
import PopoverMenuItem from '@shared/components/PopoverMenuItem.vue'
import type { ProxyStatus } from '../types'

interface Props {
  proxy: ProxyStatus
  showSource?: boolean
  showActions?: boolean
  deleting?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showSource: false,
  showActions: false,
  deleting: false,
})

defineEmits<{
  click: [proxy: ProxyStatus]
  edit: [proxy: ProxyStatus]
  delete: [proxy: ProxyStatus]
  toggle: [proxy: ProxyStatus, enabled: boolean]
}>()

const displaySource = computed(() => {
  return props.proxy.source === 'store' ? 'store' : 'config'
})

const localDisplay = computed(() => {
  if (props.proxy.plugin) return `plugin:${props.proxy.plugin}`
  return props.proxy.local_addr || ''
})

const statusClass = computed(() => {
  switch (props.proxy.status) {
    case 'running':
      return 'running'
    case 'error':
      return 'error'
    case 'disabled':
      return 'disabled'
    default:
      return 'waiting'
  }
})
</script>

<style scoped lang="scss">
.proxy-card {
  background: $color-bg-primary;
  border: 1px solid $color-border-lighter;
  border-radius: $radius-md;
  padding: 14px 20px;
  cursor: pointer;
  transition: all $transition-medium;

  &:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
    border-color: $color-border;
  }

  &.has-error {
    border-color: rgba(245, 108, 108, 0.3);
  }
}

.card-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: $spacing-lg;
}

.card-left {
  @include flex-column;
  gap: $spacing-sm;
  flex: 1;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.proxy-name {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
}

.type-tag {
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  padding: 2px 8px;
  border-radius: 4px;
  background: $color-bg-muted;
  color: $color-text-secondary;
}

.card-address {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: $font-size-sm;
  color: $color-text-muted;
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}



.card-right {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-shrink: 0;
}

.source-label {
  font-size: $font-size-xs;
  color: $color-text-light;
}


.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}



@include mobile {
  .card-main {
    flex-direction: column;
    align-items: stretch;
    gap: $spacing-sm;
  }
  .card-right {
    justify-content: space-between;
  }
  .card-address {
    word-break: break-all;
  }
}
</style>
