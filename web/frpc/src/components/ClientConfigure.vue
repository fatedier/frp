<template>
  <div>
    <el-row id="head">
      <el-button type="primary" @click="fetchData">Refresh</el-button>
      <el-button type="primary" @click="uploadConfig">Upload</el-button>
    </el-row>
    <el-input
      type="textarea"
      autosize
      v-model="textarea"
      placeholder="frpc configure file, can not be empty..."
    ></el-input>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

let textarea = ref('')

const fetchData = () => {
  fetch('/api/config', { credentials: 'include' })
    .then((res) => {
      return res.text()
    })
    .then((text) => {
      textarea.value = text
    })
    .catch(() => {
      ElMessage({
        showClose: true,
        message: 'Get configure content from frpc failed!',
        type: 'warning',
      })
    })
}

const uploadConfig = () => {
  ElMessageBox.confirm(
    'This operation will upload your frpc configure file content and hot reload it, do you want to continue?',
    'Notice',
    {
      confirmButtonText: 'Yes',
      cancelButtonText: 'No',
      type: 'warning',
    }
  )
    .then(() => {
      if (textarea.value == '') {
        ElMessage({
          message: 'Configure content can not be empty!',
          type: 'warning',
        })
        return
      }

      fetch('/api/config', {
        credentials: 'include',
        method: 'PUT',
        body: textarea.value,
      })
        .then(() => {
          fetch('/api/reload', { credentials: 'include' })
            .then(() => {
              ElMessage({
                type: 'success',
                message: 'Success',
              })
            })
            .catch((err) => {
              ElMessage({
                showClose: true,
                message: 'Reload frpc configure file error, ' + err,
                type: 'warning',
              })
            })
        })
        .catch(() => {
          ElMessage({
            showClose: true,
            message: 'Put config to frpc and hot reload failed!',
            type: 'warning',
          })
        })
    })
    .catch(() => {
      ElMessage({
        message: 'Canceled',
        type: 'info',
      })
    })
}

fetchData()
</script>

<style>
#head {
  margin-bottom: 30px;
}
</style>
