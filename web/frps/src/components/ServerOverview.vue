<template>
  <div>
    <el-row>
      <el-col :md="12">
        <div class="source">
          <el-form
            label-position="left"
            label-width="160px"
            class="server_info"
          >
            <el-form-item label="Version">
              <span>{{ data.version }}</span>
            </el-form-item>
            <el-form-item label="BindPort">
              <span>{{ data.bind_port }}</span>
            </el-form-item>
            <el-form-item label="BindUdpPort">
              <span>{{ data.bind_udp_port }}</span>
            </el-form-item>
            <el-form-item label="Http Port">
              <span>{{ data.vhost_http_port }}</span>
            </el-form-item>
            <el-form-item label="Https Port">
              <span>{{ data.vhost_https_port }}</span>
            </el-form-item>
            <el-form-item label="Subdomain Host">
              <span>{{ data.subdomain_host }}</span>
            </el-form-item>
            <el-form-item label="Max PoolCount">
              <span>{{ data.max_pool_count }}</span>
            </el-form-item>
            <el-form-item label="Max Ports Per Client">
              <span>{{ data.max_ports_per_client }}</span>
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

let data = ref({
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
  proxy_counts: 0,
})

const fetchData = () => {
  fetch('../api/serverinfo', { credentials: 'include' })
    .then((res) => res.json())
    .then((json) => {
      data.value.version = json.version
      data.value.bind_port = json.bind_port
      data.value.bind_udp_port = json.bind_udp_port
      if (data.value.bind_udp_port == '0') {
        data.value.bind_udp_port = 'disable'
      }
      data.value.vhost_http_port = json.vhost_http_port
      if (data.value.vhost_http_port == '0') {
        data.value.vhost_http_port = 'disable'
      }
      data.value.vhost_https_port = json.vhost_https_port
      if (data.value.vhost_https_port == '0') {
        data.value.vhost_https_port = 'disable'
      }
      data.value.subdomain_host = json.subdomain_host
      data.value.max_pool_count = json.max_pool_count
      data.value.max_ports_per_client = json.max_ports_per_client
      if (data.value.max_ports_per_client == '0') {
        data.value.max_ports_per_client = 'no limit'
      }
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
  border: 1px solid #eaeefb;
  border-radius: 4px;
  transition: 0.2s;
  padding: 24px;
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
