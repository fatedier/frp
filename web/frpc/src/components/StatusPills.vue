<template>
  <div class="status-pills">
    <button
      v-for="pill in pills"
      :key="pill.status"
      class="pill"
      :class="{ active: modelValue === pill.status, [pill.status || 'all']: true }"
      @click="emit('update:modelValue', pill.status)"
    >
      {{ pill.label }} {{ pill.count }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  items: Array<{ status: string }>
  modelValue: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const pills = computed(() => {
  const counts = { running: 0, error: 0, waiting: 0 }
  for (const item of props.items) {
    const s = item.status as keyof typeof counts
    if (s in counts) {
      counts[s]++
    }
  }
  return [
    { status: '', label: 'All', count: props.items.length },
    { status: 'running', label: 'Running', count: counts.running },
    { status: 'error', label: 'Error', count: counts.error },
    { status: 'waiting', label: 'Waiting', count: counts.waiting },
  ]
})
</script>

<style scoped lang="scss">
.status-pills {
  display: flex;
  gap: $spacing-sm;
}

.pill {
  border: none;
  border-radius: 12px;
  padding: $spacing-xs $spacing-md;
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  cursor: pointer;
  background: $color-bg-muted;
  color: $color-text-secondary;
  transition: all $transition-fast;
  white-space: nowrap;

  &:hover {
    opacity: 0.85;
  }

  &.active {
    &.all {
      background: $color-bg-muted;
      color: $color-text-secondary;
    }

    &.running {
      background: rgba(103, 194, 58, 0.1);
      color: #67c23a;
    }

    &.error {
      background: rgba(245, 108, 108, 0.1);
      color: #f56c6c;
    }

    &.waiting {
      background: rgba(230, 162, 60, 0.1);
      color: #e6a23c;
    }
  }
}

@include mobile {
  .status-pills {
    overflow-x: auto;
    flex-wrap: nowrap;
    scrollbar-width: none;
    -ms-overflow-style: none;

    &::-webkit-scrollbar {
      display: none;
    }
  }
}
</style>
