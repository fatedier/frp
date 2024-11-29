<template>
  <div>
    <el-row>
      <el-col :md="24">
        <div>
          <el-table :data="status" stripe style="width: 100%" :default-sort="{ prop: 'type', order: 'ascending' }">
            <el-table-column prop="name" :label="t('OverView.name')" sortable></el-table-column>
            <el-table-column prop="type" :label="t('OverView.type')" width="150" sortable></el-table-column>
            <el-table-column prop="local_addr" :label="t('OverView.local_addr')" width="200" sortable></el-table-column>
            <el-table-column prop="plugin" :label="t('OverView.plugin')" width="200" sortable></el-table-column>
            <el-table-column prop="remote_addr" :label="t('OverView.remote_addr')" sortable></el-table-column>
            <el-table-column prop="status" :label="t('OverView.status')" width="150" sortable></el-table-column>
            <el-table-column prop="err" :label="t('OverView.err')"></el-table-column>
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n';
const { t } = useI18n();

let status = ref<any[]>([])

const fetchData = () => {
  fetch('/api/status', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      status.value = new Array()
      for (let key in json) {
        for (let ps of json[key]) {
          console.log(ps)
          status.value.push(ps)
        }
      }
    })
    .catch((err) => {
      ElMessage({
        showClose: true,
        message: 'Get status info from frpc failed!' + err,
        type: 'warning',
      })
    })
}
fetchData()
</script>

<style></style>
