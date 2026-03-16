<template>
  <div class="proxy-form-layout">
    <ConfigSection :readonly="readonly">
      <ProxyBaseSection v-model="form" :readonly="readonly" :editing="editing" />
      <ProxyRemoteSection
        v-if="['tcp', 'udp', 'http', 'https', 'tcpmux'].includes(form.type)"
        v-model="form" :readonly="readonly" />
      <ProxyBackendSection v-model="form" :readonly="readonly" />
    </ConfigSection>

    <ProxyAuthSection
      v-if="['http', 'tcpmux', 'stcp', 'sudp', 'xtcp'].includes(form.type)"
      v-model="form" :readonly="readonly" />
    <ProxyHttpSection v-if="form.type === 'http'" v-model="form" :readonly="readonly" />
    <ProxyTransportSection v-model="form" :readonly="readonly" />
    <ProxyHealthSection v-model="form" :readonly="readonly" />
    <ProxyLoadBalanceSection v-model="form" :readonly="readonly" />
    <ProxyNatSection v-if="form.type === 'xtcp'" v-model="form" :readonly="readonly" />
    <ProxyMetadataSection v-model="form" :readonly="readonly" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ProxyFormData } from '../../types'
import ConfigSection from '../ConfigSection.vue'
import ProxyBaseSection from './ProxyBaseSection.vue'
import ProxyRemoteSection from './ProxyRemoteSection.vue'
import ProxyBackendSection from './ProxyBackendSection.vue'
import ProxyAuthSection from './ProxyAuthSection.vue'
import ProxyHttpSection from './ProxyHttpSection.vue'
import ProxyTransportSection from './ProxyTransportSection.vue'
import ProxyHealthSection from './ProxyHealthSection.vue'
import ProxyLoadBalanceSection from './ProxyLoadBalanceSection.vue'
import ProxyNatSection from './ProxyNatSection.vue'
import ProxyMetadataSection from './ProxyMetadataSection.vue'

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
