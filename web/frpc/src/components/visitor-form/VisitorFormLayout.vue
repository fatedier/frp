<template>
  <div class="visitor-form-layout">
    <ConfigSection :readonly="readonly">
      <VisitorBaseSection v-model="form" :readonly="readonly" :editing="editing" />
    </ConfigSection>
    <VisitorConnectionSection v-model="form" :readonly="readonly" />
    <VisitorTransportSection v-model="form" :readonly="readonly" />
    <VisitorXtcpSection v-if="form.type === 'xtcp'" v-model="form" :readonly="readonly" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { VisitorFormData } from '../../types'
import ConfigSection from '../ConfigSection.vue'
import VisitorBaseSection from './VisitorBaseSection.vue'
import VisitorConnectionSection from './VisitorConnectionSection.vue'
import VisitorTransportSection from './VisitorTransportSection.vue'
import VisitorXtcpSection from './VisitorXtcpSection.vue'

const props = withDefaults(defineProps<{
  modelValue: VisitorFormData
  readonly?: boolean
  editing?: boolean
}>(), { readonly: false, editing: false })

const emit = defineEmits<{ 'update:modelValue': [value: VisitorFormData] }>()

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>
