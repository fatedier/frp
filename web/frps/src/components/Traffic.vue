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
  mounted() {
    this.initData()
  },
  methods: {
    async initData() {
      const json = await this.$fetch(`traffic/${this.proxyName}`)
      if (!json) return

      DrawProxyTrafficChart(this.proxyName, json.traffic_in, json.traffic_out)
    }
  }
}
</script>
