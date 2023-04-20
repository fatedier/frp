<template>
  <ProxyView :proxies="proxies" proxyType="http" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { HTTPProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<HTTPProxy[]>([])

const fetchData = () => {
  let vhost_http_port: number
  let subdomain_host: string
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      vhost_http_port = json.vhost_http_port
      subdomain_host = json.subdomain_host
      if (vhost_http_port == null || vhost_http_port == 0) {
        return
      }
      fetch('../api/proxy/http', { credentials: 'include' })
        .then((res) => {
          return res.json()
        })
        .then((json) => {
          for (let proxyStats of json.proxies) {
            proxies.value.push(
              new HTTPProxy(proxyStats, vhost_http_port, subdomain_host)
            )
          }
        })
    })
}
fetchData()
</script>

<style></style>
