<template>
  <!-- Edit mode: use el-form-item for validation -->
  <el-form-item v-if="!readonly" :label="label" :prop="prop" :class="($attrs.class as string)">
    <!-- text -->
    <el-input
      v-if="type === 'text'"
      :model-value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <!-- number -->
    <el-input
      v-else-if="type === 'number'"
      :model-value="modelValue != null ? String(modelValue) : ''"
      :placeholder="placeholder"
      :disabled="disabled"
      @update:model-value="handleNumberInput($event)"
    />
    <!-- switch -->
    <div v-else-if="type === 'switch'" class="config-field-switch-wrap">
      <el-switch
        :model-value="modelValue"
        :disabled="disabled"
        size="small"
        @update:model-value="$emit('update:modelValue', $event)"
      />
      <span v-if="tip" class="config-field-switch-tip">{{ tip }}</span>
    </div>
    <!-- select -->
    <PopoverMenu
      v-else-if="type === 'select'"
      :model-value="modelValue"
      :display-value="selectDisplayValue"
      :disabled="disabled"
      :width="selectWidth"
      selectable
      full-width
      filterable
      :filter-placeholder="placeholder || 'Select...'"
      @update:model-value="$emit('update:modelValue', $event)"
    >
      <template #default="{ filterText }">
        <PopoverMenuItem
          v-for="opt in filteredOptions(filterText)"
          :key="opt.value"
          :value="opt.value"
        >
          {{ opt.label }}
        </PopoverMenuItem>
      </template>
    </PopoverMenu>
    <!-- password -->
    <el-input
      v-else-if="type === 'password'"
      :model-value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      type="password"
      show-password
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <!-- kv -->
    <KeyValueEditor
      v-else-if="type === 'kv'"
      :model-value="modelValue"
      :key-placeholder="keyPlaceholder"
      :value-placeholder="valuePlaceholder"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <!-- tags (string array) -->
    <StringListEditor
      v-else-if="type === 'tags'"
      :model-value="modelValue || []"
      :placeholder="placeholder"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <div v-if="tip && type !== 'switch'" class="config-field-tip">{{ tip }}</div>
  </el-form-item>

  <!-- Readonly mode: plain display -->
  <div v-else class="config-field-readonly" :class="($attrs.class as string)">
    <div class="config-field-label">{{ label }}</div>
    <!-- switch readonly -->
    <el-switch
      v-if="type === 'switch'"
      :model-value="modelValue"
      disabled
      size="small"
    />
    <!-- kv readonly -->
    <KeyValueEditor
      v-else-if="type === 'kv'"
      :model-value="modelValue || []"
      :key-placeholder="keyPlaceholder"
      :value-placeholder="valuePlaceholder"
      readonly
    />
    <!-- tags readonly -->
    <StringListEditor
      v-else-if="type === 'tags'"
      :model-value="modelValue || []"
      readonly
    />
    <!-- text/number/select/password readonly -->
    <el-input
      v-else
      :model-value="displayValue"
      disabled
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import KeyValueEditor from './KeyValueEditor.vue'
import StringListEditor from './StringListEditor.vue'
import PopoverMenu from '@shared/components/PopoverMenu.vue'
import PopoverMenuItem from '@shared/components/PopoverMenuItem.vue'

const props = withDefaults(
  defineProps<{
    label: string
    type?: 'text' | 'number' | 'switch' | 'select' | 'password' | 'kv' | 'tags'
    readonly?: boolean
    modelValue?: any
    placeholder?: string
    disabled?: boolean
    tip?: string
    prop?: string
    options?: Array<{ label: string; value: string | number }>
    min?: number
    max?: number
    keyPlaceholder?: string
    valuePlaceholder?: string
  }>(),
  {
    type: 'text',
    readonly: false,
    modelValue: undefined,
    placeholder: '',
    disabled: false,
    tip: '',
    prop: '',
    options: () => [],
    min: undefined,
    max: undefined,
    keyPlaceholder: 'Key',
    valuePlaceholder: 'Value',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const handleNumberInput = (val: string) => {
  if (val === '') {
    emit('update:modelValue', undefined)
    return
  }
  const num = Number(val)
  if (!isNaN(num)) {
    let clamped = num
    if (props.min != null && clamped < props.min) clamped = props.min
    if (props.max != null && clamped > props.max) clamped = props.max
    emit('update:modelValue', clamped)
  }
}

const selectDisplayValue = computed(() => {
  const opt = props.options.find((o) => o.value === props.modelValue)
  return opt ? opt.label : ''
})

const selectWidth = computed(() => {
  return Math.max(160, ...props.options.map((o) => o.label.length * 10 + 60))
})

const filteredOptions = (filterText: string) => {
  if (!filterText) return props.options
  const lower = filterText.toLowerCase()
  return props.options.filter((o) => o.label.toLowerCase().includes(lower))
}

const displayValue = computed(() => {
  if (props.modelValue == null || props.modelValue === '') return '—'
  if (props.type === 'select') {
    const opt = props.options.find((o) => o.value === props.modelValue)
    return opt ? opt.label : String(props.modelValue)
  }
  if (props.type === 'password') {
    return props.modelValue ? '••••••' : '—'
  }
  return String(props.modelValue)
})
</script>

<style scoped>
.config-field-switch-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  width: 100%;
}

.config-field-switch-tip {
  font-size: 12px;
  color: var(--color-text-muted);
}

.config-field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.config-field-readonly {
  margin-bottom: 16px;
}

.config-field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin-bottom: 6px;
  line-height: 1;
}

.config-field-readonly :deep(*) {
  cursor: default !important;
}

.config-field-readonly :deep(.el-input.is-disabled .el-input__wrapper) {
  background: var(--color-bg-tertiary);
  box-shadow: 0 0 0 1px var(--color-border-lighter) inset;
}

.config-field-readonly :deep(.el-input.is-disabled .el-input__inner) {
  color: var(--color-text-primary);
  -webkit-text-fill-color: var(--color-text-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.config-field-readonly :deep(.el-switch.is-disabled) {
  opacity: 1;
}
</style>
