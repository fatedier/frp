import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const router = vi.hoisted(() => ({ push: vi.fn() }))
vi.mock('vue-router', () => ({ useRouter: () => router }))

import StatCard from '../src/components/StatCard.vue'

const stubs = {
  'el-card': defineComponent({
    template: '<div class="el-card-stub" @click="$emit(\'click\', $event)"><slot /></div>',
    emits: ['click'],
  }),
  'el-icon': defineComponent({ template: '<span class="el-icon-stub"><slot /></span>' }),
}

describe('StatCard', () => {
  it('shows its content and does not navigate without a destination', async () => {
    router.push.mockClear()
    const wrapper = mount(StatCard, {
      props: { label: 'Clients', value: 3, subtitle: 'active' },
      global: { stubs },
    })

    expect(wrapper.find('.stat-value').text()).toBe('3')
    expect(wrapper.find('.stat-label').text()).toBe('Clients')
    expect(wrapper.find('.stat-subtitle').text()).toBe('active')
    expect(wrapper.find('.arrow-icon').exists()).toBe(false)

    await wrapper.find('.stat-card').trigger('click')

    expect(router.push).not.toHaveBeenCalled()
  })

  it('navigates only when a destination is provided', async () => {
    router.push.mockClear()
    const wrapper = mount(StatCard, {
      props: { label: 'Proxies', value: '12', type: 'proxies', to: '/proxies' },
      global: { stubs },
    })

    await wrapper.find('.stat-card').trigger('click')

    expect(router.push).toHaveBeenCalledWith('/proxies')
    expect(wrapper.find('.arrow-icon').exists()).toBe(true)
  })
})
