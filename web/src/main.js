import Vue from 'vue'
import '@/icons'
import '@/styles/index.scss'

import App from './App.vue'
import router from './router'
import store from '@/store'
import 'whatwg-fetch'

import '@/utils/element-ui'

import fetch from '@/utils/fetch'
Vue.prototype.$fetch = fetch

Vue.config.productionTip = false

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
