<template>
  <div class="configure-page">
    <div class="page-header">
      <div class="title-section">
        <h1 class="page-title">Config</h1>
      </div>
    </div>

    <div class="editor-header">
      <div class="header-left">
        <a
          href="https://github.com/fatedier/frp#configuration-files"
          target="_blank"
          class="docs-link"
        >
          <el-icon><Link /></el-icon>
          Documentation
        </a>
      </div>
      <div class="header-actions">
        <ActionButton @click="handleUpload">Update & Reload</ActionButton>
      </div>
    </div>

    <div class="editor-wrapper">
      <el-input
        type="textarea"
        :autosize="false"
        v-model="configContent"
        placeholder="# frpc configuration file content...

serverAddr = &quot;127.0.0.1&quot;
serverPort = 7000"
        class="code-editor"
      ></el-input>
    </div>

    <ConfirmDialog
      v-model="confirmVisible"
      title="Confirm Update"
      message="This operation will update your frpc configuration and reload it. Do you want to continue?"
      confirm-text="Update"
      :loading="uploading"
      :is-mobile="isMobile"
      @confirm="doUpload"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import { useClientStore } from '../stores/client'
import ActionButton from '@shared/components/ActionButton.vue'
import ConfirmDialog from '@shared/components/ConfirmDialog.vue'
import { useResponsive } from '../composables/useResponsive'

const { isMobile } = useResponsive()
const clientStore = useClientStore()
const configContent = ref('')

const fetchData = async () => {
  try {
    await clientStore.fetchConfig()
    configContent.value = clientStore.config
  } catch (err: any) {
    ElMessage({
      showClose: true,
      message: 'Get configuration failed: ' + err.message,
      type: 'warning',
    })
  }
}

const confirmVisible = ref(false)
const uploading = ref(false)

const handleUpload = () => {
  confirmVisible.value = true
}

const doUpload = async () => {
  if (!configContent.value.trim()) {
    ElMessage.warning('Configuration content cannot be empty!')
    return
  }

  uploading.value = true
  try {
    await clientStore.saveConfig(configContent.value)
    await clientStore.reload()
    ElMessage.success('Configuration updated and reloaded successfully')
    confirmVisible.value = false
  } catch (err: any) {
    ElMessage.error('Update failed: ' + err.message)
  } finally {
    uploading.value = false
  }
}

fetchData()
</script>

<style scoped lang="scss">
.configure-page {
  height: 100%;
  overflow: hidden;
  padding: $spacing-xl 40px;
  max-width: 960px;
  margin: 0 auto;
  @include flex-column;
  gap: $spacing-sm;
}

.editor-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}


.page-header {
  @include flex-column;
  gap: $spacing-sm;
  margin-bottom: $spacing-sm;
}


.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.docs-link {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  color: $color-text-muted;
  text-decoration: none;
  font-size: $font-size-sm;
  transition: color $transition-fast;

  &:hover {
    color: $color-text-primary;
  }
}

.code-editor {
  height: 100%;

  :deep(.el-textarea__inner) {
    height: 100% !important;
    overflow-y: auto;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: $font-size-sm;
    line-height: 1.6;
    padding: $spacing-lg;
    border-radius: $radius-md;
    background: $color-bg-tertiary;
    border: 1px solid $color-border-light;
    resize: none;

    &:focus {
      border-color: $color-text-light;
      box-shadow: none;
    }
  }
}

@include mobile {
  .configure-page {
    padding: $spacing-xl $spacing-lg;
  }

  .header-left {
    justify-content: space-between;
  }

  .header-actions {
    justify-content: flex-end;
  }
}
</style>
