import { createRouter, createWebHashHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import Overview from '../views/Overview.vue'
import ClientConfigure from '../views/ClientConfigure.vue'
import ProxyEdit from '../views/ProxyEdit.vue'
import VisitorEdit from '../views/VisitorEdit.vue'
import { listStoreProxies } from '../api/frpc'

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
      meta: { requiresStore: true },
    },
    {
      path: '/proxies/:name/edit',
      name: 'ProxyEdit',
      component: ProxyEdit,
      meta: { requiresStore: true },
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
  ],
})

const isStoreEnabled = async () => {
  try {
    await listStoreProxies()
    return true
  } catch (err: any) {
    if (err?.status === 404) {
      return false
    }
    return true
  }
}

router.beforeEach(async (to) => {
  if (!to.matched.some((record) => record.meta.requiresStore)) {
    return true
  }

  const enabled = await isStoreEnabled()
  if (enabled) {
    return true
  }

  ElMessage.warning(
    'Store is disabled. Enable Store in frpc config to create or edit store entries.',
  )
  return { name: 'Overview' }
})

export default router
