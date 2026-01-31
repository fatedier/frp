<template>
  <div id="app">
    <header class="header">
      <div class="header-content">
        <div class="header-top">
          <div class="brand-section">
            <div class="logo-wrapper">
              <LogoIcon class="logo-icon" />
            </div>
            <span class="divider">/</span>
            <span class="brand-name">frp</span>
            <span class="badge server-badge">Server</span>
            <span class="badge" v-if="currentRouteName">{{
              currentRouteName
            }}</span>
          </div>

          <div class="header-controls">
            <a
              class="github-link"
              href="https://github.com/fatedier/frp"
              target="_blank"
              aria-label="GitHub"
            >
              <GitHubIcon class="github-icon" />
            </a>
            <el-switch
              v-model="isDark"
              inline-prompt
              :active-icon="Moon"
              :inactive-icon="Sunny"
              class="theme-switch"
            />
          </div>
        </div>

        <nav class="nav-bar">
          <router-link to="/" class="nav-link" active-class="active"
            >Overview</router-link
          >
          <router-link to="/clients" class="nav-link" active-class="active"
            >Clients</router-link
          >
          <router-link
            to="/proxies"
            class="nav-link"
            :class="{ active: route.path.startsWith('/proxies') }"
            >Proxies</router-link
          >
        </nav>
      </div>
    </header>

    <main id="content">
      <router-view></router-view>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useDark } from '@vueuse/core'
import { Moon, Sunny } from '@element-plus/icons-vue'
import GitHubIcon from './assets/icons/github.svg?component'
import LogoIcon from './assets/icons/logo.svg?component'

const route = useRoute()
const isDark = useDark()

const currentRouteName = computed(() => {
  if (route.path === '/') return 'Overview'
  if (route.path.startsWith('/clients')) return 'Clients'
  if (route.path.startsWith('/proxies')) return 'Proxies'
  return ''
})
</script>

<style>
:root {
  --header-height: 112px;
  --header-bg: rgba(255, 255, 255, 0.8);
  --header-border: #eaeaea;
  --text-primary: #000;
  --text-secondary: #666;
  --hover-bg: #f5f5f5;
  --active-link: #000;
}

html.dark {
  --header-bg: rgba(0, 0, 0, 0.8);
  --header-border: #333;
  --text-primary: #fff;
  --text-secondary: #888;
  --hover-bg: #1a1a1a;
  --active-link: #fff;
}

body {
  margin: 0;
  font-family:
    -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue',
    Arial, sans-serif;
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--el-bg-color-page);
}

.header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--header-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--header-border);
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 40px;
}

.header-top {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.brand-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-wrapper {
  display: flex;
  align-items: center;
}

.logo-icon {
  width: 32px;
  height: 32px;
}

.divider {
  color: var(--header-border);
  font-size: 24px;
  font-weight: 200;
}

.brand-name {
  font-weight: 600;
  font-size: 18px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.badge {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--hover-bg);
  padding: 2px 8px;
  border-radius: 99px;
  border: 1px solid var(--header-border);
}

.badge.server-badge {
  background: linear-gradient(135deg, #3b82f6 0%, #06b6d4 100%);
  color: white;
  border: none;
  font-weight: 500;
}

html.dark .badge.server-badge {
  background: linear-gradient(135deg, #60a5fa 0%, #22d3ee 100%);
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.github-link {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--text-primary);
  transition: background 0.2s;
  background: transparent;
  border: 1px solid transparent;
  cursor: pointer;
}

.github-icon {
  width: 18px;
  height: 18px;
}

.github-link:hover {
  background: var(--hover-bg);
  border-color: var(--header-border);
}

.theme-switch {
  --el-switch-on-color: #2c2c3a;
  --el-switch-off-color: #f2f2f2;
  --el-switch-border-color: var(--header-border);
}

html.dark .theme-switch {
  --el-switch-off-color: #333;
}

.theme-switch .el-switch__core .el-switch__inner .el-icon {
  color: #909399 !important;
}

.nav-bar {
  height: 48px;
  display: flex;
  align-items: center;
  gap: 24px;
}

.nav-link {
  text-decoration: none;
  font-size: 14px;
  color: var(--text-secondary);
  padding: 8px 0;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.nav-link:hover {
  color: var(--text-primary);
}

.nav-link.active {
  color: var(--active-link);
  border-bottom-color: var(--active-link);
}

#content {
  flex: 1;
  width: 100%;
  padding: 40px;
  max-width: 1200px;
  margin: 0 auto;
  box-sizing: border-box;
}

@media (max-width: 768px) {
  .header-content {
    padding: 0 20px;
  }

  #content {
    padding: 20px;
  }
}
</style>
