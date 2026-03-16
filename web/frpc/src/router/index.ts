import { createRouter, createWebHashHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import ClientConfigure from '../views/ClientConfigure.vue'
import ProxyEdit from '../views/ProxyEdit.vue'
import VisitorEdit from '../views/VisitorEdit.vue'
import { useProxyStore } from '../stores/proxy'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      redirect: '/proxies',
    },
    {
      path: '/proxies',
      name: 'ProxyList',
      component: () => import('../views/ProxyList.vue'),
    },
    {
      path: '/proxies/detail/:name',
      name: 'ProxyDetail',
      component: () => import('../views/ProxyDetail.vue'),
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
      component: () => import('../views/VisitorList.vue'),
    },
    {
      path: '/visitors/detail/:name',
      name: 'VisitorDetail',
      component: () => import('../views/VisitorDetail.vue'),
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
