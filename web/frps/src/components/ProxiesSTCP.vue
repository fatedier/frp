<template>
  <ProxyView :proxies="proxies" proxyType="stcp" @refresh="fetchData"/>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { STCPProxy } from '../utils/proxy.js'
import ProxyView from './ProxyView.vue'

let proxies = ref<STCPProxy[]>([])

const fetchData = () => {
  fetch('../api/proxy/stcp', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      proxies.value = []
      for (let proxyStats of json.proxies) {
        proxies.value.push(new STCPProxy(proxyStats))
      }
    })
}
fetchData()
</script>

<style></style>
