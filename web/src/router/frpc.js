import AdminLayout from '@/components/AdminLayout'
const routes = [
  {
    path: '/frpc/config',
    component: AdminLayout,
    meta: {
      icon: 'config'
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
    path: '/frpc',
    component: AdminLayout,
    meta: {
      icon: 'dashboard'
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
  }
]

export default routes
