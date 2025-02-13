import { createRouter, createWebHashHistory } from 'vue-router'
import Overview from '../components/Overview.vue'
import ClientConfigure from '../components/ClientConfigure.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'Overview',
      component: Overview,
    },
    {
      path: '/configure',
      name: 'ClientConfigure',
      component: ClientConfigure,
    },
  ],
})

export default router
