<template>
  <el-form label-position="left" label-width="auto" inline class="proxy-table-expand">
    <el-form-item :label="t('OverView.Expand.Name')">
      <span>{{ row.name }}</span>
    </el-form-item>
    <el-form-item :label="t('OverView.Expand.Type')">
      <span>{{ row.type }}</span>
    </el-form-item>
    <el-form-item :label="t('OverView.Expand.Encryption')">
      <span>{{ row.encryption }}</span>
    </el-form-item>
    <el-form-item :label="t('OverView.Expand.Compression')">
      <span>{{ row.compression }}</span>
    </el-form-item>
    <el-form-item :label="t('OverView.Expand.LastStart')">
      <span>{{ row.lastStartTime }}</span>
    </el-form-item>
    <el-form-item :label="t('OverView.Expand.LastClose')">
      <span>{{ row.lastCloseTime }}</span>
    </el-form-item>

    <div v-if="proxyType === 'http' || proxyType === 'https'">
      <el-form-item :label="t('OverView.Expand.Domains')">
        <span>{{ row.customDomains }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.SubDomain')">
        <span>{{ row.subdomain }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.locations')">
        <span>{{ row.locations }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.HostRewrite')">
        <span>{{ row.hostHeaderRewrite }}</span>
      </el-form-item>
    </div>
    <div v-else-if="proxyType === 'tcpmux'">
      <el-form-item :label="t('OverView.Expand.Multiplexer')">
        <span>{{ row.multiplexer }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.RouteByHTTPUser')">
        <span>{{ row.routeByHTTPUser }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.Domains')">
        <span>{{ row.customDomains }}</span>
      </el-form-item>
      <el-form-item :label="t('OverView.Expand.SubDomain')">
        <span>{{ row.subdomain }}</span>
      </el-form-item>
    </div>
    <div v-else>
      <el-form-item :label="t('OverView.Expand.Addr')">
        <span>{{ row.addr }}</span>
      </el-form-item>
    </div>
  </el-form>

  <div v-if="row.annotations && row.annotations.size > 0">
    <el-divider />
    <el-text class="title-text" size="large">{{ t("OverView.Expand.Annotations") }}</el-text>
    <ul>
      <li v-for="item in annotationsArray()">
        <span class="annotation-key">{{ item.key }}</span>
        <span>{{ item.value }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
const { t } = useI18n();

const props = defineProps<{
  row: any
  proxyType: string
}>()

// annotationsArray returns an array of key-value pairs from the annotations map.
const annotationsArray = (): Array<{ key: string; value: string }> => {
  const array: Array<{ key: string; value: any }> = [];
  if (props.row.annotations) {
    props.row.annotations.forEach((value: any, key: string) => {
      array.push({ key, value });
    });
  }
  return array;
}
</script>

<style>
ul {
  list-style-type: none;
  padding: 5px;
}

ul li {
  justify-content: space-between;
  padding: 5px;
}

ul .annotation-key {
  width: 300px;
  display: inline-block;
  vertical-align: middle;
}

.title-text {
  color: #99a9bf;
}
</style>
