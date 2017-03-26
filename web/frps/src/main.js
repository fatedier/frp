import Vue from 'vue'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-default/index.css'
import './utils/less/custom.less'

import App from './App.vue'
import router from './router'

Vue.use(ElementUI)
Vue.config.productionTip = false

new Vue({
    el: '#app',
    router,
    template: '<App/>',
    components: { App }
})
