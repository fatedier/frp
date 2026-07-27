import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfigField from '../src/components/ConfigField.vue'

const InputStub = defineComponent({
  props: ['modelValue', 'disabled', 'placeholder', 'type'],
  emits: ['update:modelValue'],
  template:
    '<input :value="modelValue ?? \'\'" :disabled="disabled" :placeholder="placeholder" :type="type === \'password\' ? \'password\' : \'text\'" @input="$emit(\'update:modelValue\', $event.target.value)" />',
})

const FormItemStub = defineComponent({
  template: '<div class="form-item-stub"><slot /></div>',
})

const stubs = {
  'el-form-item': FormItemStub,
  'el-input': InputStub,
  'el-switch': defineComponent({
    props: ['modelValue', 'disabled'],
    emits: ['update:modelValue'],
    template: '<button :disabled="disabled" @click="$emit(\'update:modelValue\', !modelValue)">switch</button>',
  }),
  KeyValueEditor: defineComponent({ template: '<div class="key-value-stub" />' }),
  StringListEditor: defineComponent({ template: '<div class="string-list-stub" />' }),
  PopoverMenu: defineComponent({ template: '<div class="popover-stub"><slot /></div>' }),
  PopoverMenuItem: defineComponent({ template: '<div><slot /></div>' }),
}

describe('ConfigField', () => {
  it('clamps numeric input and emits the public model update', async () => {
    const wrapper = mount(ConfigField, {
      props: { label: 'Port', type: 'number', modelValue: 100, min: 1, max: 65535 },
      global: { stubs },
    })

    const input = wrapper.find('input')
    await input.setValue('70000')

    expect(wrapper.emitted('update:modelValue')).toContainEqual([65535])
    expect((input.element as HTMLInputElement).value).toBe('65535')
  })

  it('keeps a leading sign as an editable draft until it becomes numeric', async () => {
    const wrapper = mount(ConfigField, {
      props: { label: 'Port', type: 'number', modelValue: 100 },
      global: { stubs },
    })

    const input = wrapper.find('input')
    await wrapper.setProps({ modelValue: 200 })
    expect((input.element as HTMLInputElement).value).toBe('200')

    await input.setValue('-')

    expect(wrapper.emitted('update:modelValue')).toContainEqual([undefined])
    await wrapper.setProps({ modelValue: undefined })
    expect((input.element as HTMLInputElement).value).toBe('-')
  })

  it('renders an empty readonly field as a disabled placeholder', () => {
    const wrapper = mount(ConfigField, {
      props: { label: 'Secret', readonly: true, modelValue: '' },
      global: { stubs },
    })

    const input = wrapper.find('input').element as HTMLInputElement
    expect(wrapper.find('.config-field-label').text()).toBe('Secret')
    expect(input.disabled).toBe(true)
    expect(input.value).toBe('—')
  })
})
