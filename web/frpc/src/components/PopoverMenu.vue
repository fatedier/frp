<template>
  <div
    class="popover-menu-wrapper"
    :class="{ 'is-full-width': fullWidth }"
    ref="wrapperRef"
  >
    <el-popover
      :visible="isOpen"
      :placement="placement"
      trigger="click"
      :width="popoverWidth"
      popper-class="popover-menu-popper"
      :persistent="false"
      :hide-after="0"
      :offset="8"
      :show-arrow="false"
    >
      <template #reference>
        <div
          v-if="filterable"
          class="popover-trigger filterable-trigger"
          :class="{ 'show-clear': showClearIcon }"
          @click.stop
          @mouseenter="isHovering = true"
          @mouseleave="isHovering = false"
        >
          <el-input
            ref="filterInputRef"
            :model-value="inputValue"
            :placeholder="inputPlaceholder"
            :disabled="disabled"
            :readonly="!isOpen"
            @click="handleInputClick"
            @update:model-value="handleFilterInput"
          >
            <template #suffix>
              <el-icon
                v-if="showClearIcon"
                class="clear-icon"
                @click.stop="handleClear"
              >
                <CircleClose />
              </el-icon>
              <el-icon v-else class="arrow-icon"><ArrowDown /></el-icon>
            </template>
          </el-input>
        </div>
        <div v-else class="popover-trigger" @click.stop="toggle">
          <slot name="trigger" />
        </div>
      </template>
      <div class="popover-menu-content">
        <slot :close="close" :filter-text="filterText" />
      </div>
    </el-popover>
  </div>
</template>

<script lang="ts">
// Module-level singleton for coordinating popover menus
const popoverEventTarget = new EventTarget()
const CLOSE_ALL_EVENT = 'close-all-popovers'
</script>

<script setup lang="ts">
import {
  ref,
  computed,
  provide,
  inject,
  watch,
  onMounted,
  onUnmounted,
} from 'vue'
import { formItemContextKey, ElInput } from 'element-plus'
import { ArrowDown, CircleClose } from '@element-plus/icons-vue'

interface Props {
  width?: number
  placement?:
    | 'top'
    | 'top-start'
    | 'top-end'
    | 'bottom'
    | 'bottom-start'
    | 'bottom-end'
  modelValue?: string | number | null
  selectable?: boolean
  disabled?: boolean
  fullWidth?: boolean
  filterable?: boolean
  filterPlaceholder?: string
  displayValue?: string
  clearable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  width: 160,
  placement: 'bottom-end',
  modelValue: null,
  selectable: false,
  disabled: false,
  fullWidth: false,
  filterable: false,
  filterPlaceholder: 'Search...',
  displayValue: '',
  clearable: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number | null): void
  (e: 'filter-change', text: string): void
}>()

const elFormItem = inject(formItemContextKey, undefined)

const isOpen = ref(false)
const wrapperRef = ref<HTMLElement | null>(null)
const instanceId = Symbol()
const filterText = ref('')
const filterInputRef = ref<InstanceType<typeof ElInput> | null>(null)
const isHovering = ref(false)
const triggerWidth = ref(0)

const popoverWidth = computed(() => {
  if (props.filterable && triggerWidth.value > 0) {
    return Math.max(triggerWidth.value, props.width)
  }
  return props.width
})

const updateTriggerWidth = () => {
  if (wrapperRef.value) {
    triggerWidth.value = wrapperRef.value.offsetWidth
  }
}

const inputValue = computed(() => {
  if (isOpen.value) return filterText.value
  if (props.modelValue) return props.displayValue || ''
  return ''
})

const inputPlaceholder = computed(() => {
  if (isOpen.value) return props.filterPlaceholder
  if (!props.modelValue) return props.displayValue || props.filterPlaceholder
  return props.filterPlaceholder
})

const showClearIcon = computed(() => {
  return (
    props.clearable && props.modelValue && isHovering.value && !props.disabled
  )
})

watch(isOpen, (open) => {
  if (!open && props.filterable) {
    filterText.value = ''
    emit('filter-change', '')
  }
})

const handleInputClick = () => {
  if (props.disabled) return
  if (!isOpen.value) {
    updateTriggerWidth()
    popoverEventTarget.dispatchEvent(
      new CustomEvent(CLOSE_ALL_EVENT, { detail: instanceId }),
    )
    isOpen.value = true
  }
}

const handleFilterInput = (value: string) => {
  filterText.value = value
  emit('filter-change', value)
}

const handleClear = () => {
  emit('update:modelValue', '')
  filterText.value = ''
  emit('filter-change', '')
  elFormItem?.validate?.('change')
}

const toggle = () => {
  if (props.disabled) return
  if (!isOpen.value) {
    popoverEventTarget.dispatchEvent(
      new CustomEvent(CLOSE_ALL_EVENT, { detail: instanceId }),
    )
  }
  isOpen.value = !isOpen.value
}

const handleCloseAll = (e: Event) => {
  const customEvent = e as CustomEvent
  if (customEvent.detail !== instanceId) {
    isOpen.value = false
  }
}

const close = () => {
  isOpen.value = false
}

const select = (value: string | number) => {
  emit('update:modelValue', value)
  if (props.filterable) {
    filterText.value = ''
    emit('filter-change', '')
    filterInputRef.value?.blur()
  }
  close()
  elFormItem?.validate?.('change')
}

const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (wrapperRef.value && !wrapperRef.value.contains(target)) {
    close()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  popoverEventTarget.addEventListener(CLOSE_ALL_EVENT, handleCloseAll)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  popoverEventTarget.removeEventListener(CLOSE_ALL_EVENT, handleCloseAll)
})

provide('popoverMenu', {
  close,
  select,
  selectable: props.selectable,
  modelValue: () => props.modelValue,
})
</script>

<style scoped lang="scss">
.popover-menu-wrapper {
  display: inline-block;

  &.is-full-width {
    display: block;
    width: 100%;

    .popover-trigger {
      display: block;
      width: 100%;
    }
  }
}

.popover-trigger {
  display: inline-flex;

  &.filterable-trigger {
    display: block;
    width: 100%;

    :deep(.el-input__wrapper) {
      cursor: pointer;
    }

    :deep(.el-input__suffix) {
      cursor: pointer;
    }

    .arrow-icon {
      color: var(--el-text-color-placeholder);
      transition: transform 0.2s;
    }

    .clear-icon {
      color: var(--el-text-color-placeholder);
      transition: color 0.2s;

      &:hover {
        color: var(--el-text-color-regular);
      }
    }
  }
}

.popover-menu-content {
  padding: 4px;
}
</style>

<style lang="scss">
.popover-menu-popper {
  padding: 0 !important;
  border-radius: 12px !important;
  border: 1px solid $color-border-light !important;
  box-shadow:
    0 10px 25px -5px rgba(0, 0, 0, 0.1),
    0 8px 10px -6px rgba(0, 0, 0, 0.1) !important;
}
</style>
