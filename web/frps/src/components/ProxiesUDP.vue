<template>
  <ProxyView :proxies="proxies" proxyType="udp" @refresh="fetchData"/>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { UDPProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<UDPProxy[]>([])

const fetchData = () => {
  fetch('../api/proxy/udp', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      proxies.value = []
      for (let proxyStats of json.proxies) {
        proxies.value.push(new UDPProxy(proxyStats))
      }
    })
}
fetchData()
</script>

<style></style>
