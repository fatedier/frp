<template>
  <button
    type="button"
    class="action-button"
    :class="[variant, size, { 'is-loading': loading, 'is-danger': danger }]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <div v-if="loading" class="spinner"></div>
    <span v-if="loading && loadingText">{{ loadingText }}</span>
    <slot v-else />
  </button>
</template>

<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'outline'
  size?: 'small' | 'medium' | 'large'
  disabled?: boolean
  loading?: boolean
  loadingText?: string
  danger?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'medium',
  disabled: false,
  loading: false,
  loadingText: '',
  danger: false,
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const handleClick = (event: MouseEvent) => {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<style scoped lang="scss">
.action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  border-radius: $radius-md;
  font-weight: $font-weight-medium;
  cursor: pointer;
  transition: all $transition-fast;
  border: 1px solid transparent;
  white-space: nowrap;

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.75s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  &.small {
    padding: 5px $spacing-md;
    font-size: $font-size-sm;
  }

  &.medium {
    padding: $spacing-sm $spacing-lg;
    font-size: $font-size-md;
  }

  &.large {
    padding: 10px $spacing-xl;
    font-size: $font-size-lg;
  }

  &.primary {
    background: $color-btn-primary;
    border-color: $color-btn-primary;
    color: #fff;

    &:hover:not(:disabled) {
      background: $color-btn-primary-hover;
      border-color: $color-btn-primary-hover;
    }
  }

  &.secondary {
    background: $color-bg-hover;
    border-color: $color-border-light;
    color: $color-text-primary;

    &:hover:not(:disabled) {
      border-color: $color-border;
    }
  }

  &.outline {
    background: transparent;
    border-color: $color-border;
    color: $color-text-primary;

    &:hover:not(:disabled) {
      background: $color-bg-hover;
    }
  }

  &.is-danger {
    &.primary {
      background: $color-danger;
      border-color: $color-danger;

      &:hover:not(:disabled) {
        background: $color-danger-dark;
        border-color: $color-danger-dark;
      }
    }

    &.outline, &.secondary {
      color: $color-danger;

      &:hover:not(:disabled) {
        border-color: $color-danger;
        background: rgba(239, 68, 68, 0.08);
      }
    }
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}
</style>
