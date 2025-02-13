<template>
  <ProxyView :proxies="proxies" proxyType="https" @refresh="fetchData"/>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { HTTPSProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<HTTPSProxy[]>([])

const fetchData = () => {
  let vhostHTTPSPort: number
  let subdomainHost: string
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      vhostHTTPSPort = json.vhostHTTPSPort
      subdomainHost = json.subdomainHost
      if (vhostHTTPSPort == null || vhostHTTPSPort == 0) {
        return
      }
      fetch('../api/proxy/https', { credentials: 'include' })
        .then((res) => {
          return res.json()
        })
        .then((json) => {
          proxies.value = []
          for (let proxyStats of json.proxies) {
            proxies.value.push(
              new HTTPSProxy(proxyStats, vhostHTTPSPort, subdomainHost)
            )
          }
        })
    })
}
fetchData()
</script>

<style></style>
