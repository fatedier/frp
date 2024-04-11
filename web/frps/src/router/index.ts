import { createRouter, createWebHashHistory } from 'vue-router'
import ServerOverview from '../components/ServerOverview.vue'
import ProxiesTCP from '../components/ProxiesTCP.vue'
import ProxiesUDP from '../components/ProxiesUDP.vue'
import ProxiesHTTP from '../components/ProxiesHTTP.vue'
import ProxiesHTTPS from '../components/ProxiesHTTPS.vue'
import ProxiesTCPMux from '../components/ProxiesTCPMux.vue'
import ProxiesSTCP from '../components/ProxiesSTCP.vue'
import ProxiesSUDP from '../components/ProxiesSUDP.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'ServerOverview',
      component: ServerOverview,
    },
    {
      path: '/proxies/tcp',
      name: 'ProxiesTCP',
      component: ProxiesTCP,
    },
    {
      path: '/proxies/udp',
      name: 'ProxiesUDP',
      component: ProxiesUDP,
    },
    {
      path: '/proxies/http',
      name: 'ProxiesHTTP',
      component: ProxiesHTTP,
    },
    {
      path: '/proxies/https',
      name: 'ProxiesHTTPS',
      component: ProxiesHTTPS,
    },
    {
      path: '/proxies/tcpmux',
      name: 'ProxiesTCPMux',
      component: ProxiesTCPMux,
    },
    {
      path: '/proxies/stcp',
      name: 'ProxiesSTCP',
      component: ProxiesSTCP,
    },
    {
      path: '/proxies/sudp',
      name: 'ProxiesSUDP',
      component: ProxiesSUDP,
    },
  ],
})

export default router
