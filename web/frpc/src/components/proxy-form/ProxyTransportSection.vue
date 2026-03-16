<template>
  <ConfigSection title="Transport" collapsible :readonly="readonly"
    :has-value="form.useEncryption || form.useCompression || !!form.bandwidthLimit || (!!form.bandwidthLimitMode && form.bandwidthLimitMode !== 'client') || !!form.proxyProtocolVersion">
    <div class="field-row two-col">
      <ConfigField label="Use Encryption" type="switch" v-model="form.useEncryption" :readonly="readonly" />
      <ConfigField label="Use Compression" type="switch" v-model="form.useCompression" :readonly="readonly" />
    </div>
    <div class="field-row three-col">
      <ConfigField label="Bandwidth Limit" type="text" v-model="form.bandwidthLimit" placeholder="1MB" tip="e.g., 1MB, 500KB" :readonly="readonly" />
      <ConfigField label="Bandwidth Limit Mode" type="select" v-model="form.bandwidthLimitMode"
        :options="[{ label: 'Client', value: 'client' }, { label: 'Server', value: 'server' }]" :readonly="readonly" />
      <ConfigField label="Proxy Protocol Version" type="select" v-model="form.proxyProtocolVersion"
        :options="[{ label: 'None', value: '' }, { label: 'v1', value: 'v1' }, { label: 'v2', value: 'v2' }]" :readonly="readonly" />
    </div>
  </ConfigSection>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ProxyFormData } from '../../types'
import ConfigSection from '../ConfigSection.vue'
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
