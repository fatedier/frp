<template>
  <div class="config-section-card">
    <!-- Collapsible: header is a separate clickable area -->
    <template v-if="collapsible">
      <div
        v-if="title"
        class="section-header clickable"
        @click="handleToggle"
      >
        <h3 class="section-title">{{ title }}</h3>
        <div class="section-header-right">
          <span v-if="readonly && !hasValue" class="not-configured-badge">
            Not configured
          </span>
          <el-icon v-if="canToggle" class="collapse-arrow" :class="{ expanded }">
            <ArrowDown />
          </el-icon>
        </div>
      </div>
      <div class="collapse-wrapper" :class="{ expanded }">
        <div class="collapse-inner">
          <div class="section-body">
            <slot />
          </div>
        </div>
      </div>
    </template>

    <!-- Non-collapsible: title and content in one area -->
    <template v-else>
      <div class="section-body">
        <h3 v-if="title" class="section-title section-title-inline">{{ title }}</h3>
        <slot />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'

const props = withDefaults(
  defineProps<{
    title?: string
    collapsible?: boolean
    readonly?: boolean
    hasValue?: boolean
  }>(),
  {
    title: '',
    collapsible: false,
    readonly: false,
    hasValue: true,
  },
)

const computeInitial = () => {
  if (!props.collapsible) return true
  return props.hasValue
}

const expanded = ref(computeInitial())

// Only auto-expand when hasValue goes from false to true (async data loaded)
// Never auto-collapse — don't override user interaction
watch(
  () => props.hasValue,
  (newVal, oldVal) => {
    if (newVal && !oldVal && props.collapsible) {
      expanded.value = true
    }
  },
)

const canToggle = computed(() => {
  if (!props.collapsible) return false
  if (props.readonly && !props.hasValue) return false
  return true
})

const handleToggle = () => {
  if (canToggle.value) {
    expanded.value = !expanded.value
  }
}
</script>

<style scoped lang="scss">
.config-section-card {
  background: var(--el-bg-color);
  border: 1px solid var(--color-border-lighter);
  border-radius: 12px;
  margin-bottom: 16px;
  overflow: hidden;
}

/* Collapsible header */
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 20px;
}

.section-header.clickable {
  cursor: pointer;
  transition: background 0.15s;
}

.section-header.clickable:hover {
  background: var(--color-bg-hover);
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

/* Inline title for non-collapsible sections */
.section-title-inline {
  margin-bottom: 16px;
}

.section-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.not-configured-badge {
  font-size: 11px;
  color: var(--color-text-light);
  background: var(--color-bg-muted);
  padding: 2px 8px;
  border-radius: 4px;
}

.collapse-arrow {
  transition: transform 0.3s;
  color: var(--color-text-muted);
}

.collapse-arrow.expanded {
  transform: rotate(-180deg);
}

/* Grid-based collapse animation */
.collapse-wrapper {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.25s ease;
}

.collapse-wrapper.expanded {
  grid-template-rows: 1fr;
}

.collapse-inner {
  overflow: hidden;
}

.section-body {
  padding: 20px 20px 12px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-body :deep(.el-form-item) {
  margin-bottom: 0;
}

.section-body :deep(.config-field-readonly) {
  margin-bottom: 0;
}

@include mobile {
  .section-body {
    padding: 16px;
  }
}
</style>
