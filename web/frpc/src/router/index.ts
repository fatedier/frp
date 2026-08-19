import { createRouter, createWebHashHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import ClientConfigure from '../views/ClientConfigure.vue'
import Login from '../views/Login.vue'
import ProxyDetail from '../views/ProxyDetail.vue'
import ProxyEdit from '../views/ProxyEdit.vue'
import ProxyList from '../views/ProxyList.vue'
import VisitorDetail from '../views/VisitorDetail.vue'
import VisitorEdit from '../views/VisitorEdit.vue'
import VisitorList from '../views/VisitorList.vue'
import { useProxyStore } from '../stores/proxy'
import { checkAuthState } from '../stores/auth'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: Login,
      meta: { public: true },
    },
    {
      path: '/',
      redirect: '/proxies',
    },
    {
      path: '/proxies',
      name: 'ProxyList',
      component: ProxyList,
    },
    {
      path: '/proxies/detail/:name',
      name: 'ProxyDetail',
      component: ProxyDetail,
    },
    {
      path: '/proxies/create',
      name: 'ProxyCreate',
      component: ProxyEdit,
      meta: { requiresStore: true },
    },
    {
      path: '/proxies/:name/edit',
      name: 'ProxyEdit',
      component: ProxyEdit,
      meta: { requiresStore: true },
    },
    {
      path: '/visitors',
      name: 'VisitorList',
      component: VisitorList,
    },
    {
      path: '/visitors/detail/:name',
      name: 'VisitorDetail',
      component: VisitorDetail,
    },
    {
      path: '/visitors/create',
      name: 'VisitorCreate',
      component: VisitorEdit,
      meta: { requiresStore: true },
    },
    {
      path: '/visitors/:name/edit',
      name: 'VisitorEdit',
      component: VisitorEdit,
      meta: { requiresStore: true },
    },
    {
      path: '/config',
      name: 'ClientConfigure',
      component: ClientConfigure,
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

  if (!to.matched.some((record) => record.meta.requiresStore)) {
    return true
  }

  const proxyStore = useProxyStore()
  const enabled = await proxyStore.checkStoreEnabled()
  if (enabled) {
    return true
  }

  ElMessage.warning(
    'Store is disabled. Enable Store in frpc config to create or edit store entries.',
  )
  return { name: 'ProxyList' }
})

export default router
