<template>
  <div>
    <el-table
      :data="proxies"
      :default-sort="{ prop: 'name', order: 'ascending' }"
      style="width: 100%"
    >
      <el-table-column type="expand">
        <template #default="props">
          <el-popover
            placement="right"
            width="600"
            style="margin-left: 0px"
            trigger="click"
          >
            <template #default>
              <Traffic :proxy_name="props.row.name" />
            </template>

            <template #reference>
              <el-button
                type="primary"
                size="large"
                :name="props.row.name"
                style="margin-bottom: 10px"
                >Traffic Statistics
              </el-button>
            </template>
          </el-popover>

          <ProxyViewExpand :row="props.row" :proxyType="proxyType" />
        </template>
      </el-table-column>
      <el-table-column label="Name" prop="name" sortable> </el-table-column>
      <el-table-column label="Port" prop="port" sortable> </el-table-column>
      <el-table-column label="Connections" prop="conns" sortable>
      </el-table-column>
      <el-table-column
        label="Traffic In"
        prop="traffic_in"
        :formatter="formatTrafficIn"
        sortable
      >
      </el-table-column>
      <el-table-column
        label="Traffic Out"
        prop="traffic_out"
        :formatter="formatTrafficOut"
        sortable
      >
      </el-table-column>
      <el-table-column label="ClientVersion" prop="client_version" sortable>
      </el-table-column>
      <el-table-column label="Status" prop="status" sortable>
        <template #default="scope">
          <el-tag v-if="scope.row.status === 'online'" type="success">{{
            scope.row.status
          }}</el-tag>
          <el-tag v-else type="danger">{{ scope.row.status }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import * as Humanize from 'humanize-plus'
import type { TableColumnCtx } from 'element-plus'
import type { BaseProxy } from '../utils/proxy.js'
import ProxyViewExpand from './ProxyViewExpand.vue'

defineProps<{
  proxies: BaseProxy[]
  proxyType: string
}>()

const formatTrafficIn = (row: BaseProxy, _: TableColumnCtx<BaseProxy>) => {
  return Humanize.fileSize(row.traffic_in)
}

const formatTrafficOut = (row: BaseProxy, _: TableColumnCtx<BaseProxy>) => {
  return Humanize.fileSize(row.traffic_out)
}
</script>
