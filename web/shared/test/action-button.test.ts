import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ActionButton from '../components/ActionButton.vue'

describe('ActionButton', () => {
  it('emits click events for an enabled button', async () => {
    const wrapper = mount(ActionButton, { slots: { default: 'Save' } })

    await wrapper.trigger('click')

    expect(wrapper.emitted('click')).toHaveLength(1)
    expect(wrapper.text()).toBe('Save')
  })

  it('disables itself and shows loading text while loading', async () => {
    const wrapper = mount(ActionButton, {
      props: { loading: true, loadingText: 'Saving...' },
      slots: { default: 'Save' },
    })

    await wrapper.trigger('click')

    expect((wrapper.element as HTMLButtonElement).disabled).toBe(true)
    expect(wrapper.classes()).toContain('is-loading')
    expect(wrapper.find('.spinner').exists()).toBe(true)
    expect(wrapper.text()).toBe('Saving...')
    expect(wrapper.emitted('click')).toBeUndefined()
  })
})
