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
  {
    path: '/frps',
    component: AdminLayout,
    meta: {
      icon: 'dashboard',
      type: 'frps'
    },
    children: [
      {
        path: '',
        component: () => import('@/views/frps/Overview'),
        meta: {
          title: 'Overview'
        }
      }
    ]
  },
  {
    path: '/frps/proxies',
    component: AdminLayout,
    meta: {
      title: 'Proxies',
      icon: 'proxy',
      type: 'frps'
    },
    children: [
      {
        path: 'tcp',
        component: () => import('@/views/frps/ProxiesTcp'),
        meta: {
          title: 'TCP'
        }
      },
      {
        path: 'udp',
        component: () => import('@/views/frps/ProxiesUdp'),
        meta: {
          title: 'UDP'
        }
      },
      {
        path: 'http',
        component: () => import('@/views/frps/ProxiesHttp'),
        meta: {
          title: 'HTTP'
        }
      },
      {
        path: 'https',
        component: () => import('@/views/frps/ProxiesHttps'),
        meta: {
          title: 'HTTPS'
        }
      },
      {
        path: 'stcp',
        component: () => import('@/views/frps/ProxiesStcp'),
        meta: {
          title: 'STCP'
        }
      }
    ]
  },
  {
    path: '/frpc',
    component: AdminLayout,
    meta: {
      icon: 'dashboard',
      type: 'frpc'
    },
    children: [
      {
        path: '',
        component: () => import('@/views/frpc/Overview'),
        meta: {
          title: 'Overview'
        }
      }
    ]
  },
  {
    path: '/frpc/config',
    component: AdminLayout,
    meta: {
      icon: 'config',
      type: 'frpc'
    },
    children: [
      {
        path: '',
        component: () => import('@/views/frpc/Configure'),
        meta: {
          title: 'Configure'
        }
      }
    ]
  },
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
  const newRoutes = routes.filter(route => !route.meta || !route.meta.type || route.meta.type === process.env.VUE_APP_TYPE)
  for (const route in newRoutes) {
    if (route.children) {
      route.children = filterRoutes(route.children)
    }
  }
  return newRoutes
}

export const routes = filterRoutes(allRoutes)

export default new Router({
  routes
})
