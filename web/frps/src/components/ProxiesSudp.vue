<template>
  <ProxyView :proxies="proxies" proxyType="sudp" />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SUDPProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<SUDPProxy[]>([])

const fetchData = () => {
  fetch('../api/proxy/sudp', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      for (let proxyStats of json.proxies) {
        proxies.value.push(new SUDPProxy(proxyStats))
      }
    })
}
fetchData()
</script>

<style></style>
