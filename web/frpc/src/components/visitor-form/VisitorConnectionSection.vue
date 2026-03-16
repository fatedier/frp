<template>
  <ConfigSection title="Connection" :readonly="readonly">
    <div class="field-row two-col">
      <ConfigField label="Server Name" type="text" v-model="form.serverName" prop="serverName"
        placeholder="Name of the proxy to visit" :readonly="readonly" />
      <ConfigField label="Server User" type="text" v-model="form.serverUser"
        placeholder="Leave empty for same user" :readonly="readonly" />
    </div>
    <ConfigField label="Secret Key" type="password" v-model="form.secretKey"
      placeholder="Shared secret" :readonly="readonly" />
    <div class="field-row two-col">
      <ConfigField label="Bind Address" type="text" v-model="form.bindAddr"
        placeholder="127.0.0.1" :readonly="readonly" />
      <ConfigField label="Bind Port" type="number" v-model="form.bindPort"
        :min="bindPortMin" :max="65535" prop="bindPort" :readonly="readonly" />
    </div>
  </ConfigSection>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { VisitorFormData } from '../../types'
import ConfigSection from '../ConfigSection.vue'
import ConfigField from '../ConfigField.vue'

const props = withDefaults(defineProps<{
  modelValue: VisitorFormData
  readonly?: boolean
}>(), { readonly: false })

const emit = defineEmits<{ 'update:modelValue': [value: VisitorFormData] }>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const bindPortMin = computed(() => (form.value.type === 'sudp' ? 1 : undefined))
</script>

<style scoped lang="scss">
@use '@/assets/css/form-layout';
</style>
