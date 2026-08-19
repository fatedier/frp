<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <LogoIcon class="login-logo" />
        <span class="login-title">frp</span>
        <span class="badge login-badge">Client</span>
      </div>

      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input
            v-model="form.user"
            placeholder="Username"
            size="large"
            :prefix-icon="User"
            name="username"
            autocomplete="username"
          />
        </el-form-item>
        <el-form-item>
          <el-input
            v-model="form.password"
            type="password"
            placeholder="Password"
            size="large"
            :prefix-icon="Lock"
            name="password"
            autocomplete="current-password"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-button
          type="primary"
          size="large"
          class="login-btn"
          native-type="submit"
          :loading="loading"
        >
          Sign in
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Lock, User } from '@element-plus/icons-vue'
import LogoIcon from '../assets/icons/logo.svg?component'
import { loginRequest } from '../api/auth'
import { setAuthenticated } from '../stores/auth'

const route = useRoute()
const router = useRouter()

const form = reactive({
  user: '',
  password: '',
})
const loading = ref(false)

const handleLogin = async () => {
  if (!form.user || !form.password || loading.value) return
  loading.value = true
  try {
    await loginRequest(form.user, form.password)
    setAuthenticated(true)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.push(redirect)
  } catch {
    ElMessage.error('Invalid user or password')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.login-page {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--color-bg-secondary);
}

.login-card {
  width: 360px;
  padding: 40px 32px 32px;
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border-light);
  border-radius: 12px;
}

.login-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 32px;
}

.login-logo {
  width: 32px;
  height: 32px;
}

.login-title {
  font-weight: 600;
  font-size: 20px;
  color: var(--color-text-primary);
  letter-spacing: -0.5px;
}

.login-badge {
  background: linear-gradient(135deg, #3b82f6 0%, #06b6d4 100%);
  color: white;
}

.login-btn {
  width: 100%;
  margin-top: 8px;
}
</style>
