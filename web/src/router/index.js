import Vue from 'vue'
import Router from 'vue-router'
import AdminLayout from '@/components/AdminLayout'

Vue.use(Router)

const allRoutes = [
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

// filter routes recursively
const filterRoutes = function(routes) {
  const newRoutes = routes.filter(route => !!route)
  for (const route in newRoutes) {
    if (route.children) {
      route.children = filterRoutes(route.children)
    }
  }
  return newRoutes
}

export const routes = filterRoutes(allRoutes)
console.log('allRoutes', allRoutes, routes)

const router = new Router({
  routes
})

router.beforeEach(async (to, from, next) => {
  document.title = `${to.meta.title || 'dashboard'} - frp`
  next()
})

export default router
