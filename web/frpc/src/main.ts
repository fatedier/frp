import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'

import './assets/css/var.css'
import './assets/css/dark.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
