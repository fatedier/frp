<template>
  <BaseDialog
    v-model="visible"
    :title="title"
    width="400px"
    :close-on-click-modal="false"
    :append-to-body="true"
    :is-mobile="isMobile"
  >
    <p class="confirm-message">{{ message }}</p>
    <template #footer>
      <div class="dialog-footer">
        <ActionButton variant="outline" @click="handleCancel">
          {{ cancelText }}
        </ActionButton>
        <ActionButton
          :danger="danger"
          :loading="loading"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </ActionButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseDialog from './BaseDialog.vue'
import ActionButton from '@shared/components/ActionButton.vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    message: string
    confirmText?: string
    cancelText?: string
    danger?: boolean
    loading?: boolean
    isMobile?: boolean
  }>(),
  {
    confirmText: 'Confirm',
    cancelText: 'Cancel',
    danger: false,
    loading: false,
    isMobile: false,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  visible.value = false
  emit('cancel')
}
</script>

<style scoped lang="scss">
.confirm-message {
  margin: 0;
  font-size: $font-size-md;
  color: $color-text-secondary;
  line-height: 1.6;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
}
</style>
