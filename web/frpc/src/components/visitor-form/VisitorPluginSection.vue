<template>
  <ConfigSection title="Plugin" :readonly="readonly">
    <div class="field-row two-col">
      <ConfigField
        label="Plugin Type"
        type="select"
        v-model="form.pluginType"
        :options="pluginOptions"
        placeholder="None"
        :readonly="readonly"
      />
      <ConfigField
        v-if="form.pluginType === 'virtual_net'"
        label="Destination IP"
        type="text"
        v-model="form.pluginDestinationIP"
        prop="pluginDestinationIP"
        placeholder="10.10.10.10"
        tip="Destination address in the frp virtual network."
        :readonly="readonly"
      />
    </div>
  </ConfigSection>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { VisitorFormData } from '../../types'
import ConfigField from '../ConfigField.vue'
import ConfigSection from '../ConfigSection.vue'

const pluginOptions = [
  { label: 'None', value: '' },
  { label: 'virtual_net', value: 'virtual_net' },
]

const props = withDefaults(defineProps<{
  modelValue: VisitorFormData
  readonly?: boolean
}>(), { readonly: false })

const emit = defineEmits<{ 'update:modelValue': [value: VisitorFormData] }>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>

<style scoped lang="scss">
@use '@/assets/css/form-layout';
</style>
