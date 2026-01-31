<template>
  <div class="configure-page">
    <div class="page-header">
      <div class="title-section">
        <h1 class="page-title">Configuration</h1>
        <p class="page-subtitle">
          Edit and manage your frpc configuration file
        </p>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :xs="24" :lg="16">
        <el-card class="editor-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <div class="header-left">
                <span class="card-title">Configuration Editor</span>
                <el-tag size="small" type="success">TOML</el-tag>
              </div>
              <div class="header-actions">
                <el-tooltip content="Refresh" placement="top">
                  <el-button :icon="Refresh" circle @click="fetchData" />
                </el-tooltip>
                <el-button type="primary" :icon="Upload" @click="handleUpload">
                  Update & Reload
                </el-button>
              </div>
            </div>
          </template>

          <div class="editor-wrapper">
            <el-input
              type="textarea"
              :autosize="{ minRows: 20, maxRows: 40 }"
              v-model="configContent"
              placeholder="# frpc configuration file content...

[common]
server_addr = 127.0.0.1
server_port = 7000"
              class="code-editor"
            ></el-input>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="8">
        <el-card class="help-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">Quick Reference</span>
            </div>
          </template>
          <div class="help-content">
            <div class="help-section">
              <h4 class="help-section-title">Common Settings</h4>
              <div class="help-items">
                <div class="help-item">
                  <code>serverAddr</code>
                  <span>Server address</span>
                </div>
                <div class="help-item">
                  <code>serverPort</code>
                  <span>Server port (default: 7000)</span>
                </div>
                <div class="help-item">
                  <code>auth.token</code>
                  <span>Authentication token</span>
                </div>
              </div>
            </div>

            <div class="help-section">
              <h4 class="help-section-title">Proxy Types</h4>
              <div class="proxy-type-tags">
                <el-tag type="primary" effect="plain">TCP</el-tag>
                <el-tag type="success" effect="plain">UDP</el-tag>
                <el-tag type="warning" effect="plain">HTTP</el-tag>
                <el-tag type="danger" effect="plain">HTTPS</el-tag>
                <el-tag type="info" effect="plain">STCP</el-tag>
                <el-tag effect="plain">XTCP</el-tag>
              </div>
            </div>

            <div class="help-section">
              <h4 class="help-section-title">Example Proxy</h4>
              <pre class="code-example">
[[proxies]]
name = "web"
type = "http"
localPort = 80
customDomains = ["example.com"]</pre
              >
            </div>

            <div class="help-section">
              <a
                href="https://github.com/fatedier/frp#configuration-files"
                target="_blank"
                class="docs-link"
              >
                <el-icon><Link /></el-icon>
                View Full Documentation
              </a>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Upload, Link } from '@element-plus/icons-vue'
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
    },
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
.configure-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.title-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.page-title {
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0;
  line-height: 1.2;
}

.page-subtitle {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin: 0;
}

.editor-card,
.help-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
}

html.dark .editor-card,
html.dark .help-card {
  border-color: #3a3d5c;
  background: #27293d;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

html.dark .card-title {
  color: #e5e7eb;
}

.editor-wrapper {
  position: relative;
}

.code-editor :deep(.el-textarea__inner) {
  font-family:
    ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: 16px;
  border-radius: 8px;
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  resize: none;
}

html.dark .code-editor :deep(.el-textarea__inner) {
  background: #1e1e2d;
  border-color: #3a3d5c;
  color: #e5e7eb;
}

.code-editor :deep(.el-textarea__inner:focus) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary-light-5);
}

/* Help Card */
.help-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.help-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.help-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.help-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.help-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
  font-size: 13px;
}

html.dark .help-item {
  background: #1e1e2d;
}

.help-item code {
  font-family:
    ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.help-item span {
  color: var(--el-text-color-secondary);
  flex: 1;
}

.proxy-type-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.code-example {
  font-family:
    ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  margin: 0;
  overflow-x: auto;
  white-space: pre;
}

html.dark .code-example {
  background: #1e1e2d;
  border-color: #3a3d5c;
  color: #e5e7eb;
}

.docs-link {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-color-primary);
  text-decoration: none;
  font-size: 14px;
  font-weight: 500;
  padding: 12px 16px;
  background: var(--el-color-primary-light-9);
  border-radius: 8px;
  transition: all 0.2s;
}

.docs-link:hover {
  background: var(--el-color-primary-light-8);
}

@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-left {
    justify-content: space-between;
  }

  .header-actions {
    justify-content: flex-end;
  }
}

@media (max-width: 992px) {
  .help-card {
    margin-top: 20px;
  }
}
</style>
