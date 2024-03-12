<template>
  <el-form
    label-position="left"
    inline
    class="proxy-table-expand"
    v-if="proxyType === 'http' || proxyType === 'https'"
  >
    <el-form-item label="Name">
      <span>{{ row.name }}</span>
    </el-form-item>
    <el-form-item label="Type">
      <span>{{ row.type }}</span>
    </el-form-item>
    <el-form-item label="Domains">
      <span>{{ row.customDomains }}</span>
    </el-form-item>
    <el-form-item label="SubDomain">
      <span>{{ row.subdomain }}</span>
    </el-form-item>
    <el-form-item label="locations">
      <span>{{ row.locations }}</span>
    </el-form-item>
    <el-form-item label="HostRewrite">
      <span>{{ row.hostHeaderRewrite }}</span>
    </el-form-item>
    <el-form-item label="Encryption">
      <span>{{ row.encryption }}</span>
    </el-form-item>
    <el-form-item label="Compression">
      <span>{{ row.compression }}</span>
    </el-form-item>
    <el-form-item label="Last Start">
      <span>{{ row.lastStartTime }}</span>
    </el-form-item>
    <el-form-item label="Last Close">
      <span>{{ row.lastCloseTime }}</span>
    </el-form-item>
  </el-form>

  <el-form label-position="left" inline class="proxy-table-expand" v-else>
    <el-form-item label="Name">
      <span>{{ row.name }}</span>
    </el-form-item>
    <el-form-item label="Type">
      <span>{{ row.type }}</span>
    </el-form-item>
    <el-form-item label="Addr">
      <span>{{ row.addr }}</span>
    </el-form-item>
    <el-form-item label="Encryption">
      <span>{{ row.encryption }}</span>
    </el-form-item>
    <el-form-item label="Compression">
      <span>{{ row.compression }}</span>
    </el-form-item>
    <el-form-item label="Last Start">
      <span>{{ row.lastStartTime }}</span>
    </el-form-item>
    <el-form-item label="Last Close">
      <span>{{ row.lastCloseTime }}</span>
    </el-form-item>
  </el-form>

  <div v-if="row.annotations && row.annotations.size > 0">
  <el-divider />
  <el-text class="title-text" size="large">Annotations</el-text>
  <ul>
    <li v-for="item in annotationsArray()">
      <span class="annotation-key">{{ item.key }}</span>
      <span>{{  item.value }}</span>
    </li>
  </ul>
  </div>
</template>

<script setup lang="ts">

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
