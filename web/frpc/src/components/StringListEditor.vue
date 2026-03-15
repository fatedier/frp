<template>
  <div class="string-list-editor">
    <template v-if="readonly">
      <div v-if="!modelValue || modelValue.length === 0" class="list-empty">—</div>
      <div v-for="(item, index) in modelValue" :key="index" class="list-readonly-item">
        {{ item }}
      </div>
    </template>
    <template v-else>
      <div v-for="(item, index) in modelValue" :key="index" class="item-row">
        <el-input
          :model-value="item"
          :placeholder="placeholder"
          @update:model-value="updateItem(index, $event)"
        />
        <button class="item-remove" @click="removeItem(index)">
          <svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z" fill="currentColor"/>
          </svg>
        </button>
      </div>
      <button class="list-add-btn" @click="addItem">
        <svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M8 2a.5.5 0 01.5.5v5h5a.5.5 0 010 1h-5v5a.5.5 0 01-1 0v-5h-5a.5.5 0 010-1h5v-5A.5.5 0 018 2z" fill="currentColor"/>
        </svg>
        Add
      </button>
    </template>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string[]
    placeholder?: string
    readonly?: boolean
  }>(),
  {
    placeholder: 'Enter value',
    readonly: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const addItem = () => {
  emit('update:modelValue', [...(props.modelValue || []), ''])
}

const removeItem = (index: number) => {
  const newValue = [...props.modelValue]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}

const updateItem = (index: number, value: string) => {
  const newValue = [...props.modelValue]
  newValue[index] = value
  emit('update:modelValue', newValue)
}
</script>

<style scoped>
.string-list-editor {
  width: 100%;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.item-row .el-input {
  flex: 1;
}

.item-remove {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s;
}

.item-remove svg {
  width: 14px;
  height: 14px;
}

.item-remove:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.list-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.list-add-btn svg {
  width: 13px;
  height: 13px;
}

.list-add-btn:hover {
  background: var(--color-bg-hover);
}

.list-empty {
  color: var(--color-text-muted);
  font-size: 13px;
}

.list-readonly-item {
  font-size: 13px;
  color: var(--color-text-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  padding: 2px 0;
}
</style>
