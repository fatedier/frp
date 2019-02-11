import Vue from 'vue'
// import ElementUI from 'element-ui'
import {
    Button,
    Form,
    FormItem,
    Row,
    Col,
    Table,
    TableColumn,
    Menu,
    MenuItem,
    MessageBox,
    Message,
    Input
} from 'element-ui'
import lang from 'element-ui/lib/locale/lang/en'
import locale from 'element-ui/lib/locale'
import 'element-ui/lib/theme-chalk/index.css'
import './utils/less/custom.less'

import App from './App.vue'
import router from './router'
import 'whatwg-fetch'

locale.use(lang)

Vue.use(Button)
Vue.use(Form)
Vue.use(FormItem)
Vue.use(Row)
Vue.use(Col)
Vue.use(Table)
Vue.use(TableColumn)
Vue.use(Menu)
Vue.use(MenuItem)
Vue.use(Input)

Vue.prototype.$msgbox = MessageBox;
Vue.prototype.$confirm = MessageBox.confirm
Vue.prototype.$message = Message

//Vue.use(ElementUI)

Vue.config.productionTip = false

new Vue({
    el: '#app',
    router,
    template: '<App/>',
    components: { App }
})
