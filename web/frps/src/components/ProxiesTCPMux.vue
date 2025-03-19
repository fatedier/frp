<template>
  <ProxyView :proxies="proxies" proxyType="tcpmux" @refresh="fetchData" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { TCPMuxProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<TCPMuxProxy[]>([])

const fetchData = () => {
  let tcpmuxHTTPConnectPort: number
  let subdomainHost: string
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      tcpmuxHTTPConnectPort = json.tcpmuxHTTPConnectPort
      subdomainHost = json.subdomainHost

      fetch('../api/proxy/tcpmux', { credentials: 'include' })
        .then((res) => {
          return res.json()
        })
        .then((json) => {
          proxies.value = []
          for (let proxyStats of json.proxies) {
            proxies.value.push(new TCPMuxProxy(proxyStats, tcpmuxHTTPConnectPort, subdomainHost))
          }
        })
    })
}
fetchData()
</script>

<style></style>
