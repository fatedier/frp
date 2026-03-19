<template>
  <div class="visitor-detail-page">
    <!-- Fixed Header -->
    <div class="detail-top">
      <nav class="breadcrumb">
        <router-link to="/visitors" class="breadcrumb-link">Visitors</router-link>
        <span class="breadcrumb-sep">&rsaquo;</span>
        <span class="breadcrumb-current">{{ visitorName }}</span>
      </nav>

      <template v-if="visitor">
        <div class="detail-header">
          <div>
            <h2 class="detail-title">{{ visitor.name }}</h2>
            <p class="header-subtitle">Type: {{ visitor.type.toUpperCase() }}</p>
          </div>
          <div v-if="isStore" class="header-actions">
            <ActionButton variant="outline" size="small" @click="handleEdit">
              Edit
            </ActionButton>
          </div>
        </div>
      </template>
    </div>

    <div v-if="notFound" class="not-found">
      <p class="empty-text">Visitor not found</p>
      <p class="empty-hint">The visitor "{{ visitorName }}" does not exist.</p>
      <ActionButton variant="outline" @click="router.push('/visitors')">
        Back to Visitors
      </ActionButton>
    </div>

    <div v-else-if="visitor" v-loading="loading" class="detail-content">
      <VisitorFormLayout
        v-if="formData"
        :model-value="formData"
        readonly
      />
    </div>

    <div v-else v-loading="loading" class="loading-area"></div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import ActionButton from '@shared/components/ActionButton.vue'
import VisitorFormLayout from '../components/visitor-form/VisitorFormLayout.vue'
import { getVisitorConfig, getStoreVisitor } from '../api/frpc'
import type { VisitorDefinition, VisitorFormData } from '../types'
import { storeVisitorToForm } from '../types'

const route = useRoute()
const router = useRouter()

const visitorName = route.params.name as string
const visitor = ref<VisitorDefinition | null>(null)
const loading = ref(true)
const notFound = ref(false)
const isStore = ref(false)

onMounted(async () => {
  try {
    const config = await getVisitorConfig(visitorName)
    visitor.value = config

    // Check if visitor is from the store (for Edit/Delete buttons)
    try {
      await getStoreVisitor(visitorName)
      isStore.value = true
    } catch {
      // Not a store visitor — Edit/Delete not available
    }
  } catch (err: any) {
    if (err?.status === 404 || err?.response?.status === 404) {
      notFound.value = true
    } else {
      notFound.value = true
      ElMessage.error('Failed to load visitor: ' + err.message)
    }
  } finally {
    loading.value = false
  }
})

const formData = computed<VisitorFormData | null>(() => {
  if (!visitor.value) return null
  return storeVisitorToForm(visitor.value)
})

const handleEdit = () => {
  router.push('/visitors/' + encodeURIComponent(visitorName) + '/edit')
}

</script>

<style scoped lang="scss">
.visitor-detail-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 960px;
  margin: 0 auto;
}

.detail-top {
  flex-shrink: 0;
  padding: $spacing-xl 24px 0;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 24px 160px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  font-size: $font-size-md;
  margin-bottom: $spacing-lg;
}

.breadcrumb-link {
  color: $color-text-secondary;
  text-decoration: none;

  &:hover {
    color: $color-text-primary;
  }
}

.breadcrumb-sep {
  color: $color-text-light;
}

.breadcrumb-current {
  color: $color-text-primary;
  font-weight: $font-weight-medium;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: $spacing-xl;
}

.detail-title {
  margin: 0;
  font-size: 22px;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
  margin-bottom: $spacing-sm;
}

.header-subtitle {
  font-size: $font-size-sm;
  color: $color-text-muted;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: $spacing-sm;
}

.not-found,
.loading-area {
  text-align: center;
  padding: 60px $spacing-xl;
}

.empty-text {
  font-size: $font-size-lg;
  font-weight: $font-weight-medium;
  color: $color-text-secondary;
  margin: 0 0 $spacing-xs;
}

.empty-hint {
  font-size: $font-size-sm;
  color: $color-text-muted;
  margin: 0 0 $spacing-lg;
}

@include mobile {
  .detail-top {
    padding: $spacing-xl $spacing-lg 0;
  }

  .detail-content {
    padding: 0 $spacing-lg $spacing-xl;
  }

  .detail-header {
    flex-direction: column;
    gap: $spacing-md;
  }
}
</style>
