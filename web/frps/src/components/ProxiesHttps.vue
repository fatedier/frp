<template>
  <ProxyView :proxies="proxies" proxyType="https" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { HTTPSProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<HTTPSProxy[]>([])

const fetchData = () => {
  let vhost_https_port: number
  let subdomain_host: string
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      vhost_https_port = json.vhost_https_port
      subdomain_host = json.subdomain_host
      if (vhost_https_port == null || vhost_https_port == 0) {
        return
      }
      fetch('../api/proxy/https', { credentials: 'include' })
        .then((res) => {
          return res.json()
        })
        .then((json) => {
          for (let proxyStats of json.proxies) {
            proxies.value.push(
              new HTTPSProxy(proxyStats, vhost_https_port, subdomain_host)
            )
          }
        })
    })
}
fetchData()
</script>

<style></style>
