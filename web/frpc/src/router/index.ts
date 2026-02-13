import { createRouter, createWebHashHistory } from 'vue-router'
import Overview from '../views/Overview.vue'
import ClientConfigure from '../views/ClientConfigure.vue'
import ProxyEdit from '../views/ProxyEdit.vue'
import VisitorEdit from '../views/VisitorEdit.vue'

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
    {
      path: '/proxies/create',
      name: 'ProxyCreate',
      component: ProxyEdit,
    },
    {
      path: '/proxies/:name/edit',
      name: 'ProxyEdit',
      component: ProxyEdit,
    },
    {
      path: '/visitors/create',
      name: 'VisitorCreate',
      component: VisitorEdit,
    },
    {
      path: '/visitors/:name/edit',
      name: 'VisitorEdit',
      component: VisitorEdit,
    },
  ],
})

export default router
