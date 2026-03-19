<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="dialogWidth"
    :destroy-on-close="destroyOnClose"
    :close-on-click-modal="closeOnClickModal"
    :close-on-press-escape="closeOnPressEscape"
    :append-to-body="appendToBody"
    :top="dialogTop"
    :fullscreen="isMobile"
    class="base-dialog"
    :class="{ 'mobile-dialog': isMobile }"
  >
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    width?: string
    destroyOnClose?: boolean
    closeOnClickModal?: boolean
    closeOnPressEscape?: boolean
    appendToBody?: boolean
    top?: string
    isMobile?: boolean
  }>(),
  {
    width: '480px',
    destroyOnClose: true,
    closeOnClickModal: true,
    closeOnPressEscape: true,
    appendToBody: false,
    top: '15vh',
    isMobile: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const dialogWidth = computed(() => {
  if (props.isMobile) return '100%'
  return props.width
})

const dialogTop = computed(() => {
  if (props.isMobile) return '0'
  return props.top
})
</script>

<style lang="scss">
.base-dialog.el-dialog {
  border-radius: 16px;

  .el-dialog__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 8px;
    min-height: 42px;
    margin: 0;
    position: relative;

    &::after {
      content: "";
      position: absolute;
      bottom: 0;
      left: 8px;
      right: 8px;
      height: 1px;
      background: $color-border-lighter;
    }
  }

  .el-dialog__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
  }

  .el-dialog__body {
    padding: 16px 8px;
  }

  .el-dialog__headerbtn {
    position: static;
    width: 32px;
    height: 32px;
    @include flex-center;
    border-radius: $radius-sm;
    transition: background $transition-fast;

    &:hover {
      background: $color-bg-hover;
    }
  }

  .el-dialog__footer {
    padding: 8px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  &.mobile-dialog {
    border-radius: 0;
    margin: 0;
    height: 100%;
    max-height: 100dvh;
    display: flex;
    flex-direction: column;

    .el-dialog__body {
      flex: 1;
      overflow-y: auto;
      padding: 16px 12px;
    }

    .el-dialog__footer {
      padding: 8px 12px;
      padding-bottom: calc(8px + env(safe-area-inset-bottom));
    }
  }
}
</style>
