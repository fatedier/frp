import Vue from 'vue'
import ElementUI from 'element-ui'
import locale from 'element-ui/lib/locale/lang/en'
import 'element-ui/lib/theme-default/index.css'
import './utils/less/custom.less'

import App from './App.vue'
import router from './router'
import 'whatwg-fetch'

Vue.use(ElementUI, { locale })
Vue.config.productionTip = false

new Vue({
    el: '#app',
    router,
    template: '<App/>',
    components: { App }
})
