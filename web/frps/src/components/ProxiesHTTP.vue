<template>
  <ProxyView :proxies="proxies" proxyType="http" @refresh="fetchData"/>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { HTTPProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<HTTPProxy[]>([])

const fetchData = () => {
  let vhostHTTPPort: number
  let subdomainHost: string
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      vhostHTTPPort = json.vhostHTTPPort
      subdomainHost = json.subdomainHost
      if (vhostHTTPPort == null || vhostHTTPPort == 0) {
        return
      }
      fetch('../api/proxy/http', { credentials: 'include' })
        .then((res) => {
          return res.json()
        })
        .then((json) => {
          proxies.value = []
          for (let proxyStats of json.proxies) {
            proxies.value.push(
              new HTTPProxy(proxyStats, vhostHTTPPort, subdomainHost)
            )
          }
        })
    })
}
fetchData()
</script>

<style></style>
