<template>
  <div>
    <el-row>
      <el-col :md="24">
        <div>
          <el-table
            :data="status"
            stripe
            style="width: 100%"
            :default-sort="{ prop: 'type', order: 'ascending' }"
          >
            <el-table-column prop="name" label="name"></el-table-column>
            <el-table-column
              prop="type"
              label="type"
              width="150"
            ></el-table-column>
            <el-table-column
              prop="local_addr"
              label="local address"
              width="200"
            ></el-table-column>
            <el-table-column
              prop="plugin"
              label="plugin"
              width="200"
            ></el-table-column>
            <el-table-column
              prop="remote_addr"
              label="remote address"
            ></el-table-column>
            <el-table-column
              prop="status"
              label="status"
              width="150"
            ></el-table-column>
            <el-table-column prop="err" label="info"></el-table-column>
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ElMessage } from "element-plus";

let status = ref<any[]>([]);

const fetchData = () => {
  fetch("/api/status", { credentials: "include" })
    .then((res) => {
      return res.json();
    })
    .then((json) => {
      status.value = new Array();
      for (let s of json.tcp) {
        status.value.push(s);
      }
      for (let s of json.udp) {
        status.value.push(s);
      }
      for (let s of json.http) {
        status.value.push(s);
      }
      for (let s of json.https) {
        status.value.push(s);
      }
      for (let s of json.stcp) {
        status.value.push(s);
      }
      for (let s of json.sudp) {
        status.value.push(s);
      }
      for (let s of json.xtcp) {
        status.value.push(s);
      }
    })
    .catch(() => {
      ElMessage({
        showClose: true,
        message: "Get status info from frpc failed!",
        type: "warning",
      });
    });
};
fetchData();
</script>

<style></style>
