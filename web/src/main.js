import Vue from 'vue'
import ElementUI from 'element-ui'
import lang from 'element-ui/lib/locale/lang/en'
import locale from 'element-ui/lib/locale'
import 'element-ui/lib/theme-chalk/index.css'
import '@/icons'
import '@/styles/index.scss'

import App from './App.vue'
import router from './router'
import store from '@/store'
import 'whatwg-fetch'

locale.use(lang)

Vue.use(ElementUI)

import fetch from '@/utils/fetch'
Vue.prototype.$fetch = fetch

Vue.config.productionTip = false

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
