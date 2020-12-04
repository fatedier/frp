import Vue from 'vue'
import SvgIcon from '@/components/SvgIcon' // svg component

// register globally
Vue.component('SvgIcon', SvgIcon)

const req = require.context('./svg', false, /\.svg$/)
const requireAll = requireContext => requireContext.keys().map(requireContext)
requireAll(req)
