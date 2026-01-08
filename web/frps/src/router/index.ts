import { createRouter, createWebHashHistory } from 'vue-router'
import ServerOverview from '../views/ServerOverview.vue'
import Clients from '../views/Clients.vue'
import Proxies from '../views/Proxies.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'ServerOverview',
      component: ServerOverview,
    },
    {
      path: '/clients',
      name: 'Clients',
      component: Clients,
    },
    {
      path: '/proxies/:type?',
      name: 'Proxies',
      component: Proxies,
    },
  ],
})

export default router
