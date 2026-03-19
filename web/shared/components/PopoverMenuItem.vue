<template>
  <button
    class="popover-menu-item"
    :class="{
      'is-danger': danger,
      'is-selected': isSelected,
      'is-disabled': disabled,
    }"
    :disabled="disabled"
    @click="handleClick"
  >
    <span class="item-content">
      <slot />
    </span>
    <el-icon v-if="isSelected" class="check-icon">
      <Check />
    </el-icon>
  </button>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { Check } from '@element-plus/icons-vue'

interface Props {
  danger?: boolean
  disabled?: boolean
  value?: string | number
}

const props = withDefaults(defineProps<Props>(), {
  danger: false,
  disabled: false,
  value: undefined,
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const popoverMenu = inject<{
  close: () => void
  select: (value: string | number) => void
  selectable: boolean
  modelValue: () => string | number | null
}>('popoverMenu')

const isSelected = computed(() => {
  if (!popoverMenu?.selectable || props.value === undefined) return false
  return popoverMenu.modelValue() === props.value
})

const handleClick = () => {
  if (props.disabled) return

  if (popoverMenu?.selectable && props.value !== undefined) {
    popoverMenu.select(props.value)
  } else {
    emit('click')
    popoverMenu?.close()
  }
}
</script>

<style scoped lang="scss">
.popover-menu-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: $color-text-secondary;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s ease;
  text-align: left;
  white-space: nowrap;

  &:hover:not(.is-disabled) {
    background: $color-bg-hover;
  }

  &.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  &.is-danger {
    color: $color-danger;

    .item-content :deep(.el-icon) {
      color: $color-danger;
    }

    &:hover:not(.is-disabled) {
      background: $color-danger-light;
    }
  }

  &.is-selected {
    background: $color-bg-hover;
  }

  .item-content {
    display: flex;
    align-items: center;
    gap: 10px;
    color: inherit;

    :deep(.el-icon) {
      font-size: 16px;
      color: $color-text-light;
    }
  }

  .check-icon {
    font-size: 16px;
    color: $color-primary;
    flex-shrink: 0;
  }
}
</style>
