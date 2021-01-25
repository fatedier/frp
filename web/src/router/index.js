import Vue from 'vue'
import Router from 'vue-router'
import AdminLayout from '@/components/AdminLayout'

Vue.use(Router)

export const routes = [
  {
    path: '/',
    component: AdminLayout,
    meta: {
      hidden: true
    },
    children: [
      {
        path: '/',
        component: () => import('@/views/index')
      }
    ]
  },
  ...require(`./${process.env.VUE_APP_TYPE}`).default,
  {
    path: '/help',
    component: AdminLayout,
    children: [
      {
        path: 'https://github.com/fatedier/frp',
        meta: {
          title: 'Help',
          icon: 'help'
        }
      }
    ]
  }
]

const router = new Router({
  routes
})

router.beforeEach(async (to, from, next) => {
  document.title = `${to.meta.title || 'dashboard'} - frp`
  next()
})

export default router
