<template>
  <div :id="proxyName" style="width: 600px; height: 400px" />
</template>

<script>
import { DrawProxyTrafficChart } from '../utils/chart.js'
export default {
  props: {
    proxyName: {
      type: String,
      required: true
    }
  },
  created() {
    this.fetchData()
  },
  // watch: {
  // '$route': 'fetchData'
  // },
  methods: {
    fetchData() {
      const url = '/api/traffic/' + this.proxyName
      fetch(url, { credentials: 'include' })
        .then(res => {
          return res.json()
        })
        .then(json => {
          DrawProxyTrafficChart(this.proxyName, json.traffic_in, json.traffic_out)
        })
        .catch(err => {
          this.$message({
            showClose: true,
            message: 'Get server info from frps failed!' + err,
            type: 'warning'
          })
        })
    }
  }
}
</script>

<style>
</style>
