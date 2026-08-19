import { createRouter, createWebHashHistory } from 'vue-router'
import ServerOverview from '../views/ServerOverview.vue'
import Clients from '../views/Clients.vue'
import ClientDetail from '../views/ClientDetail.vue'
import Proxies from '../views/Proxies.vue'
import ProxyDetail from '../views/ProxyDetail.vue'
import Login from '../views/Login.vue'
import { checkAuthState } from '../stores/auth'

const router = createRouter({
  history: createWebHashHistory(),
  scrollBehavior() {
    return { top: 0 }
  },
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: Login,
      meta: { public: true },
    },
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
      path: '/clients/:key',
      name: 'ClientDetail',
      component: ClientDetail,
    },
    {
      path: '/proxies/:type?',
      name: 'Proxies',
      component: Proxies,
    },
    {
      path: '/proxy/:name',
      name: 'ProxyDetail',
      component: ProxyDetail,
    },
  ],
})

router.beforeEach(async (to) => {
  const authed = await checkAuthState()
  if (to.meta.public) {
    // Already authenticated users are sent back to the dashboard.
    if (authed && to.name === 'Login') {
      return { path: '/' }
    }
    return true
  }
  if (!authed) {
    return { name: 'Login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router
