<template>
  <div>
    <el-page-header :icon="null" style="width: 100%; margin-left: 30px; margin-bottom: 20px">
      <template #title>
        <span>{{ proxyType }}</span>
      </template>
      <template #content> </template>
      <template #extra>
        <div class="flex items-center" style="margin-right: 30px">
          <el-popconfirm :title="t('ProxiesView.ClearOffLineInfo')" @confirm="clearOfflineProxies">
            <template #reference>
              <el-button>{{ t("ProxiesView.ClearOffLine") }}</el-button>
            </template>
          </el-popconfirm>
          <el-button @click="$emit('refresh')">{{ t("ProxiesView.Refresh") }}</el-button>
        </div>
      </template>
    </el-page-header>

    <el-table :data="proxies" :default-sort="{ prop: 'name', order: 'ascending' }" style="width: 100%">
      <el-table-column type="expand">
        <template #default="props">
          <ProxyViewExpand :row="props.row" :proxyType="proxyType" />
        </template>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.name')" prop="name" sortable> </el-table-column>
      <el-table-column :label="t('ProxiesView.Port')" prop="port" sortable> </el-table-column>
      <el-table-column :label="t('ProxiesView.Connections')" prop="conns" sortable>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.Traffic_In')" prop="trafficIn" :formatter="formatTrafficIn" sortable>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.Traffic_Out')" prop="trafficOut" :formatter="formatTrafficOut" sortable>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.ClientVersion')" prop="clientVersion" sortable>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.Status.title')" prop="status" sortable>
        <template #default="scope">
          <el-tag v-if="scope.row.status === 'online'" type="success">{{
            t("ProxiesView.Status.Successinfo")
          }}</el-tag>
          <el-tag v-else type="danger">{{ scope.row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('ProxiesView.Operations.title')">
        <template #default="scope">
          <el-button type="primary" :name="scope.row.name" style="margin-bottom: 10px"
            @click="dialogVisibleName = scope.row.name; dialogVisible = true">{{ t("ProxiesView.Operations.Traffic") }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>

  <el-dialog v-model="dialogVisible" destroy-on-close="true" :title="dialogVisibleName" width="700px">
    <Traffic :proxyName="dialogVisibleName" />
  </el-dialog>
</template>

<script setup lang="ts">
import * as Humanize from 'humanize-plus'
import type { TableColumnCtx } from 'element-plus'
import type { BaseProxy } from '../utils/proxy.js'
import { ElMessage } from 'element-plus'
import ProxyViewExpand from './ProxyViewExpand.vue'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n';
const { t } = useI18n();

defineProps<{
  proxies: BaseProxy[]
  proxyType: string
}>()

const emit = defineEmits(['refresh'])

const dialogVisible = ref(false)
const dialogVisibleName = ref("")

const formatTrafficIn = (row: BaseProxy, _: TableColumnCtx<BaseProxy>) => {
  return Humanize.fileSize(row.trafficIn)
}

const formatTrafficOut = (row: BaseProxy, _: TableColumnCtx<BaseProxy>) => {
  return Humanize.fileSize(row.trafficOut)
}

const clearOfflineProxies = () => {
  fetch('../api/proxies?status=offline', {
    method: 'DELETE',
    credentials: 'include',
  })
    .then((res) => {
      if (res.ok) {
        ElMessage({
          message: 'Successfully cleared offline proxies',
          type: 'success',
        })
        emit('refresh')
      } else {
        ElMessage({
          message: 'Failed to clear offline proxies: ' + res.status + ' ' + res.statusText,
          type: 'warning',
        })
      }
    })
    .catch((err) => {
      ElMessage({
        message: 'Failed to clear offline proxies: ' + err.message,
        type: 'warning',
      })
    })
}
</script>

<style>
.el-page-header__title {
  font-size: 20px;
}
</style>
