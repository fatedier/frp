<template>
    <div :id="proxy_name" style="width: 600px;height:400px;"></div>
</template>

<script>
import {DrawProxyTrafficChart} from '../utils/chart.js'
export default {
    props: ['proxy_name'],
    created() {
        this.fetchData()
    },
    //watch: {
        //'$route': 'fetchData'
    //},
    methods: {
        fetchData() {
            let url = '/api/traffic/' + this.proxy_name
            fetch(url, {credentials: 'include'})
              .then(res => {
                return res.json()
              }).then(json => {
                DrawProxyTrafficChart(this.proxy_name, json.traffic_in, json.traffic_out)
              }).catch( err => {
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
