import { createApp } from 'vue'
import { createI18n } from 'vue-i18n';
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'

import './assets/dark.css'
import en from './assets/locales/en.json';
import zh from './assets/locales/zh.json';
const storedLocale = localStorage.getItem('i18n') || 'en';
const i18n = createI18n({
    locale: storedLocale, // 默认语言
    messages: {
        en,
        zh,
    },
});

const app = createApp(App)
app.use(router)
app.use(i18n);

app.mount('#app')
