<template>
  <div :id="proxy_name" style="width: 600px; height: 400px"></div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { DrawProxyTrafficChart } from '../utils/chart.js'

const props = defineProps<{
  proxy_name: string
}>()

const fetchData = () => {
  let url = '../api/traffic/' + props.proxy_name
  fetch(url, { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      DrawProxyTrafficChart(props.proxy_name, json.traffic_in, json.traffic_out)
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
