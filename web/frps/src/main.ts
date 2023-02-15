import { createApp } from 'vue'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'

import './assets/custom.css'

const app = createApp(App)

app.use(router)

app.mount('#app')
