<template>
  <template v-if="['tcp', 'udp'].includes(form.type)">
    <div class="field-row two-col">
      <ConfigField label="Remote Port" type="number" v-model="form.remotePort"
        :min="0" :max="65535" prop="remotePort" tip="Use 0 for random port assignment" :readonly="readonly" />
      <div></div>
    </div>
  </template>
  <template v-if="['http', 'https', 'tcpmux'].includes(form.type)">
    <div class="field-row two-col">
      <ConfigField label="Custom Domains" type="tags" v-model="form.customDomains"
        prop="customDomains" placeholder="example.com" :readonly="readonly" />
      <ConfigField v-if="form.type !== 'tcpmux'" label="Subdomain" type="text"
        v-model="form.subdomain" placeholder="test" :readonly="readonly" />
      <ConfigField v-if="form.type === 'tcpmux'" label="Multiplexer" type="select"
        v-model="form.multiplexer" :options="[{ label: 'HTTP CONNECT', value: 'httpconnect' }]" :readonly="readonly" />
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ProxyFormData } from '../../types'
import ConfigField from '../ConfigField.vue'

const props = withDefaults(defineProps<{
  modelValue: ProxyFormData
  readonly?: boolean
}>(), { readonly: false })

const emit = defineEmits<{ 'update:modelValue': [value: ProxyFormData] }>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>

<style scoped lang="scss">
@use '@/assets/css/form-layout';
</style>
