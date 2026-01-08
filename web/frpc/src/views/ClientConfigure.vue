<template>
  <div class="configure-page">
    <el-card class="main-card" shadow="never">
      <div class="toolbar-header">
        <h2 class="card-title">Client Configuration</h2>
        <div class="toolbar-actions">
          <el-tooltip content="Refresh" placement="top">
            <el-button :icon="Refresh" circle @click="fetchData" />
          </el-tooltip>
          <el-button type="primary" :icon="Upload" @click="handleUpload">Update</el-button>
        </div>
      </div>

      <div class="config-editor">
        <el-input
          type="textarea"
          :autosize="{ minRows: 10, maxRows: 30 }"
          v-model="configContent"
          placeholder="frpc configuration file content..."
          class="code-input"
        ></el-input>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Upload } from '@element-plus/icons-vue'
import { getConfig, putConfig, reloadConfig } from '../api/frpc'

const configContent = ref('')

const fetchData = async () => {
  try {
    const text = await getConfig()
    configContent.value = text
  } catch (err: any) {
    ElMessage({
      showClose: true,
      message: 'Get configuration failed: ' + err.message,
      type: 'warning',
    })
  }
}

const handleUpload = () => {
  ElMessageBox.confirm(
    'This operation will update your frpc configuration and reload it. Do you want to continue?',
    'Confirm Update',
    {
      confirmButtonText: 'Update',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  )
    .then(async () => {
      if (!configContent.value.trim()) {
        ElMessage({
          message: 'Configuration content cannot be empty!',
          type: 'warning',
        })
        return
      }

      try {
        await putConfig(configContent.value)
        await reloadConfig()
        ElMessage({
          type: 'success',
          message: 'Configuration updated and reloaded successfully',
        })
      } catch (err: any) {
        ElMessage({
          showClose: true,
          message: 'Update failed: ' + err.message,
          type: 'error',
        })
      }
    })
    .catch(() => {
        // cancelled
    })
}

fetchData()
</script>

<style scoped>
.main-card {
  border-radius: 12px;
  border: none;
}

.toolbar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding-bottom: 16px;
}

.card-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
}

.code-input {
    font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
}
</style>