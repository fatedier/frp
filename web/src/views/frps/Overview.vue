<template>
  <div>
    <el-row>
      <el-col :md="12">
        <div class="source">
          <el-form label-position="left" class="server_info">
            <el-form-item label="Version">
              <span>{{ version }}</span>
            </el-form-item>
            <el-form-item label="BindPort">
              <span>{{ bind_port }}</span>
            </el-form-item>
            <el-form-item label="BindUdpPort">
              <span>{{ bind_udp_port }}</span>
            </el-form-item>
            <el-form-item label="Http Port">
              <span>{{ vhost_http_port }}</span>
            </el-form-item>
            <el-form-item label="Https Port">
              <span>{{ vhost_https_port }}</span>
            </el-form-item>
            <el-form-item label="Subdomain Host">
              <span>{{ subdomain_host }}</span>
            </el-form-item>
            <el-form-item label="Max PoolCount">
              <span>{{ max_pool_count }}</span>
            </el-form-item>
            <el-form-item label="Max Ports Per Client">
              <span>{{ max_ports_per_client }}</span>
            </el-form-item>
            <el-form-item label="HeartBeat Timeout">
              <span>{{ heart_beat_timeout }}</span>
            </el-form-item>
            <el-form-item label="Client Counts">
              <span>{{ client_counts }}</span>
            </el-form-item>
            <el-form-item label="Current Connections">
              <span>{{ cur_conns }}</span>
            </el-form-item>
            <el-form-item label="Proxy Counts">
              <span>{{ proxy_counts }}</span>
            </el-form-item>
          </el-form>
        </div>
      </el-col>
      <el-col :md="12">
        <div id="traffic" style="width: 400px; height: 250px; margin-bottom: 30px"></div>
        <div id="proxies" style="width: 400px; height: 250px"></div>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import { DrawTrafficChart, DrawProxyChart } from '@/utils/chart.js'
export default {
  data() {
    return {
      version: '',
      bind_port: '',
      bind_udp_port: '',
      vhost_http_port: '',
      vhost_https_port: '',
      subdomain_host: '',
      max_pool_count: '',
      max_ports_per_client: '',
      heart_beat_timeout: '',
      client_counts: '',
      cur_conns: '',
      proxy_counts: ''
    }
  },
  computed: {
    serverInfo() {
      return this.$store.state.server.serverInfo
    }
  },
  async mounted() {
    await this.$store.dispatch('server/fetchServerInfo')
    this.initData()
  },
  methods: {
    initData() {
      if (!this.serverInfo) return
      console.log('serverInfo', this.serverInfo)

      this.version = this.serverInfo.version
      this.bind_port = this.serverInfo.bind_port
      this.bind_udp_port = this.serverInfo.bind_udp_port
      if (this.bind_udp_port === 0) {
        this.bind_udp_port = 'disable'
      }
      this.vhost_http_port = this.serverInfo.vhost_http_port
      if (this.vhost_http_port === 0) {
        this.vhost_http_port = 'disable'
      }
      this.vhost_https_port = this.serverInfo.vhost_https_port
      if (this.vhost_https_port === 0) {
        this.vhost_https_port = 'disable'
      }
      this.subdomain_host = this.serverInfo.subdomain_host
      this.max_pool_count = this.serverInfo.max_pool_count
      this.max_ports_per_client = this.serverInfo.max_ports_per_client
      if (this.max_ports_per_client === 0) {
        this.max_ports_per_client = 'no limit'
      }
      this.heart_beat_timeout = this.serverInfo.heart_beat_timeout
      this.client_counts = this.serverInfo.client_counts
      this.cur_conns = this.serverInfo.cur_conns
      this.proxy_counts = 0
      if (this.serverInfo.proxy_type_count != null) {
        if (this.serverInfo.proxy_type_count.tcp != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.tcp
        }
        if (this.serverInfo.proxy_type_count.udp != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.udp
        }
        if (this.serverInfo.proxy_type_count.http != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.http
        }
        if (this.serverInfo.proxy_type_count.https != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.https
        }
        if (this.serverInfo.proxy_type_count.stcp != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.stcp
        }
        if (this.serverInfo.proxy_type_count.xtcp != null) {
          this.proxy_counts += this.serverInfo.proxy_type_count.xtcp
        }
      }
      DrawTrafficChart('traffic', this.serverInfo.total_traffic_in, this.serverInfo.total_traffic_out)
      DrawProxyChart('proxies', this.serverInfo)
    }
  }
}
</script>

<style>
.source {
  border: 1px solid #eaeefb;
  border-radius: 4px;
  transition: 0.2s;
  padding: 24px;
}

.server_info {
  margin-left: 40px;
  font-size: 0px;
}

.server_info label {
  width: 150px;
  color: #99a9bf;
}

.server_info .el-form-item {
  margin-right: 0;
  margin-bottom: 0;
  width: 100%;
}
</style>
