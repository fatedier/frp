import { createApp } from 'vue'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'

import './assets/css/custom.css'
import './assets/css/dark.css'

const app = createApp(App)

app.use(router)

app.mount('#app')
