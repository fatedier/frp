import Vue from 'vue'
import Router from 'vue-router'
import AdminLayout from '@/components/AdminLayout'

Vue.use(Router)

export const routes = [
  {
    path: '/',
    component: AdminLayout,
    meta: {
      icon: 'dashboard'
    },
    children: [
      {
        path: '/',
        component: () => import('@/views/Overview'),
        name: 'Overview',
        meta: {
          title: 'Overview'
        }
      }
    ]
  },
  {
    path: '/proxies',
    component: AdminLayout,
    meta: {
      title: 'Proxies',
      icon: 'proxy'
    },
    children: [
      {
        path: 'tcp',
        component: () => import('@/views/ProxiesTcp'),
        name: 'ProxiesTcp',
        meta: {
          title: 'TCP'
        }
      },
      {
        path: 'udp',
        component: () => import('@/views/ProxiesUdp'),
        name: 'ProxiesUdp',
        meta: {
          title: 'UDP'
        }
      },
      {
        path: 'http',
        component: () => import('@/views/ProxiesHttp'),
        name: 'ProxiesHttp',
        meta: {
          title: 'HTTP'
        }
      },
      {
        path: 'https',
        component: () => import('@/views/ProxiesHttps'),
        name: 'ProxiesHttps',
        meta: {
          title: 'HTTPS'
        }
      },
      {
        path: 'stcp',
        component: () => import('@/views/ProxiesStcp'),
        name: 'ProxiesStcp',
        meta: {
          title: 'STCP'
        }
      }
    ]
  },
  {
    path: 'help',
    component: AdminLayout,
    children: [
      {
        path: 'https://github.com/fatedier/frp',
        component: () => import('@/views/Overview'),
        meta: {
          title: 'Help',
          icon: 'help'
        }
      }
    ]
  }
]

export default new Router({ routes })
