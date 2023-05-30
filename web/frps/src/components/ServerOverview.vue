<template>
  <div>
    <el-row>
      <el-col :md="12">
        <div class="source">
          <el-form
            label-position="left"
            label-width="220px"
            class="server_info"
          >
            <el-form-item label="Version">
              <span>{{ data.version }}</span>
            </el-form-item>
            <el-form-item label="BindPort">
              <span>{{ data.bind_port }}</span>
            </el-form-item>
            <el-form-item label="KCP Bind Port" v-if="data.kcp_bind_port != 0">
              <span>{{ data.kcp_bind_port }}</span>
            </el-form-item>
            <el-form-item
              label="QUIC Bind Port"
              v-if="data.quic_bind_port != 0"
            >
              <span>{{ data.quic_bind_port }}</span>
            </el-form-item>
            <el-form-item label="Http Port" v-if="data.vhost_http_port != 0">
              <span>{{ data.vhost_http_port }}</span>
            </el-form-item>
            <el-form-item label="Https Port" v-if="data.vhost_https_port != 0">
              <span>{{ data.vhost_https_port }}</span>
            </el-form-item>
            <el-form-item
              label="TCPMux HTTPConnect Port"
              v-if="data.tcpmux_httpconnect_port != 0"
            >
              <span>{{ data.tcpmux_httpconnect_port }}</span>
            </el-form-item>
            <el-form-item
              label="Subdomain Host"
              v-if="data.subdomain_host != ''"
            >
              <LongSpan :content="data.subdomain_host" :length="30"></LongSpan>
            </el-form-item>
            <el-form-item label="Max PoolCount">
              <span>{{ data.max_pool_count }}</span>
            </el-form-item>
            <el-form-item label="Max Ports Per Client">
              <span>{{ data.max_ports_per_client }}</span>
            </el-form-item>
            <el-form-item label="Allow Ports" v-if="data.allow_ports_str != ''">
              <LongSpan :content="data.allow_ports_str" :length="30"></LongSpan>
            </el-form-item>
            <el-form-item label="TLS Only" v-if="data.tls_only === true">
              <span>{{ data.tls_only }}</span>
            </el-form-item>
            <el-form-item label="HeartBeat Timeout">
              <span>{{ data.heart_beat_timeout }}</span>
            </el-form-item>
            <el-form-item label="Client Counts">
              <span>{{ data.client_counts }}</span>
            </el-form-item>
            <el-form-item label="Current Connections">
              <span>{{ data.cur_conns }}</span>
            </el-form-item>
            <el-form-item label="Proxy Counts">
              <span>{{ data.proxy_counts }}</span>
            </el-form-item>
          </el-form>
        </div>
      </el-col>
      <el-col :md="12">
        <div
          id="traffic"
          style="width: 400px; height: 250px; margin-bottom: 30px"
        ></div>
        <div id="proxies" style="width: 400px; height: 250px"></div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { DrawTrafficChart, DrawProxyChart } from '../utils/chart'
import LongSpan from './LongSpan.vue'

let data = ref({
  version: '',
  bind_port: 0,
  kcp_bind_port: 0,
  quic_bind_port: 0,
  vhost_http_port: 0,
  vhost_https_port: 0,
  tcpmux_httpconnect_port: 0,
  subdomain_host: '',
  max_pool_count: 0,
  max_ports_per_client: '',
  allow_ports_str: '',
  tls_only: false,
  heart_beat_timeout: 0,
  client_counts: 0,
  cur_conns: 0,
  proxy_counts: 0,
})

const fetchData = () => {
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => res.json())
    .then((json) => {
      data.value.version = json.version
      data.value.bind_port = json.bind_port
      data.value.kcp_bind_port = json.kcp_bind_port
      data.value.quic_bind_port = json.quic_bind_port
      data.value.vhost_http_port = json.vhost_http_port
      data.value.vhost_https_port = json.vhost_https_port
      data.value.tcpmux_httpconnect_port = json.tcpmux_httpconnect_port
      data.value.subdomain_host = json.subdomain_host
      data.value.max_pool_count = json.max_pool_count
      data.value.max_ports_per_client = json.max_ports_per_client
      if (data.value.max_ports_per_client == '0') {
        data.value.max_ports_per_client = 'no limit'
      }
      data.value.allow_ports_str = json.allow_ports_str
      data.value.tls_only = json.tls_only
      data.value.heart_beat_timeout = json.heart_beat_timeout
      data.value.client_counts = json.client_counts
      data.value.cur_conns = json.cur_conns
      data.value.proxy_counts = 0
      if (json.proxy_type_count != null) {
        if (json.proxy_type_count.tcp != null) {
          data.value.proxy_counts += json.proxy_type_count.tcp
        }
        if (json.proxy_type_count.udp != null) {
          data.value.proxy_counts += json.proxy_type_count.udp
        }
        if (json.proxy_type_count.http != null) {
          data.value.proxy_counts += json.proxy_type_count.http
        }
        if (json.proxy_type_count.https != null) {
          data.value.proxy_counts += json.proxy_type_count.https
        }
        if (json.proxy_type_count.stcp != null) {
          data.value.proxy_counts += json.proxy_type_count.stcp
        }
        if (json.proxy_type_count.sudp != null) {
          data.value.proxy_counts += json.proxy_type_count.sudp
        }
        if (json.proxy_type_count.xtcp != null) {
          data.value.proxy_counts += json.proxy_type_count.xtcp
        }
      }

      // draw chart
      DrawTrafficChart('traffic', json.total_traffic_in, json.total_traffic_out)
      DrawProxyChart('proxies', json)
    })
    .catch(() => {
      ElMessage({
        showClose: true,
        message: 'Get server info from frps failed!',
        type: 'warning',
      })
    })
}
fetchData()
</script>

<style>
.source {
  border-radius: 4px;
  transition: 0.2s;
  padding-left: 24px;
  padding-right: 24px;
}

.server_info {
  margin-left: 40px;
  font-size: 0px;
}

.server_info .el-form-item__label {
  color: #99a9bf;
  height: 40px;
  line-height: 40px;
}

.server_info .el-form-item__content {
  height: 40px;
  line-height: 40px;
}

.server_info .el-form-item {
  margin-right: 0;
  margin-bottom: 0;
  width: 100%;
}
</style>
