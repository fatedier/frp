<template>
  <!-- Name / Type / Enabled -->
  <div v-if="!readonly" class="field-row three-col">
    <el-form-item label="Name" prop="name" class="field-grow">
      <el-input
        v-model="form.name"
        :disabled="editing || readonly"
        placeholder="my-proxy"
      />
    </el-form-item>
    <ConfigField
      label="Type"
      type="select"
      v-model="form.type"
      :disabled="editing"
      :options="PROXY_TYPES.map((t) => ({ label: t.toUpperCase(), value: t }))"
      prop="type"
    />
    <el-form-item label="Enabled" class="switch-field">
      <el-switch v-model="form.enabled" size="small" />
    </el-form-item>
  </div>
  <div v-else class="field-row three-col">
    <ConfigField label="Name" type="text" :model-value="form.name" readonly class="field-grow" />
    <ConfigField label="Type" type="text" :model-value="form.type.toUpperCase()" readonly />
    <ConfigField label="Enabled" type="switch" :model-value="form.enabled" readonly />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { PROXY_TYPES, type ProxyFormData } from '../../types'
import ConfigField from '../ConfigField.vue'

const props = withDefaults(defineProps<{
  modelValue: ProxyFormData
  readonly?: boolean
  editing?: boolean
}>(), { readonly: false, editing: false })

const emit = defineEmits<{ 'update:modelValue': [value: ProxyFormData] }>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>

<style scoped lang="scss">
@use '@/assets/css/form-layout';
</style>
