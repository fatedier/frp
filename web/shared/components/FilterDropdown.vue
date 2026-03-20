<template>
  <PopoverMenu
    :model-value="modelValue"
    :width="width"
    placement="bottom-start"
    selectable
    :display-value="displayLabel"
    @update:model-value="$emit('update:modelValue', $event as string)"
  >
    <template #trigger>
      <button class="filter-trigger" :class="{ 'has-value': modelValue }" :style="minWidth && !isMobile ? { minWidth: minWidth + 'px' } : undefined">
        <span class="filter-label">{{ label }}:</span>
        <span class="filter-value">{{ displayLabel }}</span>
        <el-icon class="filter-arrow"><ArrowDown /></el-icon>
      </button>
    </template>
    <PopoverMenuItem value="">{{ allLabel }}</PopoverMenuItem>
    <PopoverMenuItem
      v-for="opt in options"
      :key="opt.value"
      :value="opt.value"
    >
      {{ opt.label }}
    </PopoverMenuItem>
  </PopoverMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'
import PopoverMenu from './PopoverMenu.vue'
import PopoverMenuItem from './PopoverMenuItem.vue'

interface Props {
  modelValue: string
  label: string
  options: Array<{ label: string; value: string }>
  allLabel?: string
  width?: number
  minWidth?: number
  isMobile?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  allLabel: 'All',
  width: 150,
})

defineEmits<{
  'update:modelValue': [value: string]
}>()

const displayLabel = computed(() => {
  if (!props.modelValue) return props.allLabel
  const found = props.options.find((o) => o.value === props.modelValue)
  return found ? found.label : props.modelValue
})
</script>

<style scoped lang="scss">
.filter-trigger {
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  padding: 7px 12px;
  background: $color-bg-primary;
  border: none;
  border-radius: $radius-md;
  box-shadow: 0 0 0 1px $color-border-light inset;
  font-size: $font-size-sm;
  color: $color-text-secondary;
  cursor: pointer;
  transition: box-shadow $transition-fast;
  white-space: nowrap;

  &:hover {
    box-shadow: 0 0 0 1px $color-border inset;
  }

  &.has-value .filter-value {
    color: $color-text-primary;
  }
}

.filter-label {
  color: $color-text-muted;
  flex-shrink: 0;
}

.filter-value {
  color: $color-text-secondary;
  margin-left: auto;
}

.filter-arrow {
  font-size: 12px;
  color: $color-text-light;
  flex-shrink: 0;
}
</style>
