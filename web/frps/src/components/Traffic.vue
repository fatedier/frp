<template>
  <div :id="proxyName" style="width: 600px; height: 400px"></div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { DrawProxyTrafficChart } from '../utils/chart.js'

const props = defineProps<{
  proxyName: string
}>()

const fetchData = () => {
  let url = '../api/traffic/' + props.proxyName
  fetch(url, { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      DrawProxyTrafficChart(props.proxyName, json.trafficIn, json.trafficOut)
    })
    .catch((err) => {
      ElMessage({
        showClose: true,
        message: 'Get traffic info failed!' + err,
        type: 'warning',
      })
    })
}
fetchData()
</script>
<style></style>
