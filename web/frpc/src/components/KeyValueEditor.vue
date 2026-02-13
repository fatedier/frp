<template>
  <div class="kv-editor">
    <div v-for="(entry, index) in modelValue" :key="index" class="kv-row">
      <el-input
        :model-value="entry.key"
        :placeholder="keyPlaceholder"
        class="kv-input"
        @update:model-value="updateEntry(index, 'key', $event)"
      />
      <el-input
        :model-value="entry.value"
        :placeholder="valuePlaceholder"
        class="kv-input"
        @update:model-value="updateEntry(index, 'value', $event)"
      />
      <button class="kv-remove-btn" @click="removeEntry(index)">
        <svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M5.5 5.5A.5.5 0 016 6v6a.5.5 0 01-1 0V6a.5.5 0 01.5-.5zm2.5 0a.5.5 0 01.5.5v6a.5.5 0 01-1 0V6a.5.5 0 01.5-.5zm3 .5a.5.5 0 00-1 0v6a.5.5 0 001 0V6z"
            fill="currentColor"
          />
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M14.5 3a1 1 0 01-1 1H13v9a2 2 0 01-2 2H5a2 2 0 01-2-2V4h-.5a1 1 0 010-2H6a1 1 0 011-1h2a1 1 0 011 1h3.5a1 1 0 011 1zM4.118 4L4 4.059V13a1 1 0 001 1h6a1 1 0 001-1V4.059L11.882 4H4.118zM6 2h4v1H6V2z"
            fill="currentColor"
          />
        </svg>
      </button>
    </div>
    <button class="kv-add-btn" @click="addEntry">
      <svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M8 2a.5.5 0 01.5.5v5h5a.5.5 0 010 1h-5v5a.5.5 0 01-1 0v-5h-5a.5.5 0 010-1h5v-5A.5.5 0 018 2z" fill="currentColor" />
      </svg>
      Add
    </button>
  </div>
</template>

<script setup lang="ts">
interface KVEntry {
  key: string
  value: string
}

interface Props {
  modelValue: KVEntry[]
  keyPlaceholder?: string
  valuePlaceholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  keyPlaceholder: 'Key',
  valuePlaceholder: 'Value',
})

const emit = defineEmits<{
  'update:modelValue': [value: KVEntry[]]
}>()

const updateEntry = (index: number, field: 'key' | 'value', val: string) => {
  const updated = [...props.modelValue]
  updated[index] = { ...updated[index], [field]: val }
  emit('update:modelValue', updated)
}

const addEntry = () => {
  emit('update:modelValue', [...props.modelValue, { key: '', value: '' }])
}

const removeEntry = (index: number) => {
  const updated = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', updated)
}
</script>

<style scoped>
.kv-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.kv-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.kv-input {
  flex: 1;
}

.kv-remove-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.kv-remove-btn svg {
  width: 14px;
  height: 14px;
}

.kv-remove-btn:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

html.dark .kv-remove-btn:hover {
  background: rgba(248, 113, 113, 0.15);
  color: #f87171;
}

.kv-add-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  align-self: flex-start;
}

.kv-add-btn svg {
  width: 14px;
  height: 14px;
}

.kv-add-btn:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
</style>
