import AdminLayout from '@/components/AdminLayout'
const routes = [
  {
    path: '/frps',
    component: AdminLayout,
    meta: {
      icon: 'dashboard'
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
      icon: 'proxy'
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
  }
]

export default routes
