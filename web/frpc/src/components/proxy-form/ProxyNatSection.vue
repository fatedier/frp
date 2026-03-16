<template>
  <ConfigSection title="NAT Traversal" collapsible :readonly="readonly" :has-value="form.natTraversalDisableAssistedAddrs">
    <ConfigField label="Disable Assisted Addresses" type="switch" v-model="form.natTraversalDisableAssistedAddrs"
      tip="Only use STUN-discovered public addresses" :readonly="readonly" />
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
