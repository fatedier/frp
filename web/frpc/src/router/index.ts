import { createRouter, createWebHashHistory } from 'vue-router'
import Overview from '../views/Overview.vue'
import ClientConfigure from '../views/ClientConfigure.vue'

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
