<template>
  <div id="app">
    <header class="header">
      <div class="header-content">
        <div class="brand-section">
          <button
            v-if="isMobile"
            class="hamburger-btn"
            @click="toggleSidebar"
            aria-label="Toggle menu"
          >
            <span class="hamburger-icon">&#9776;</span>
          </button>
          <div class="logo-wrapper">
            <LogoIcon class="logo-icon" />
          </div>
          <span class="divider">/</span>
          <span class="brand-name">frp</span>
          <span class="badge server-badge">Server</span>
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
    </header>

    <div class="layout">
      <!-- Mobile overlay -->
      <div
        v-if="isMobile && sidebarOpen"
        class="sidebar-overlay"
        @click="closeSidebar"
      />

      <aside
        class="sidebar"
        :class="{ 'mobile-open': isMobile && sidebarOpen }"
      >
        <nav class="sidebar-nav">
          <router-link
            to="/"
            class="sidebar-link"
            :class="{ active: route.path === '/' }"
            @click="closeSidebar"
          >
            Overview
          </router-link>
          <router-link
            to="/clients"
            class="sidebar-link"
            :class="{ active: route.path.startsWith('/clients') }"
            @click="closeSidebar"
          >
            Clients
          </router-link>
          <router-link
            to="/proxies"
            class="sidebar-link"
            :class="{
              active:
                route.path.startsWith('/proxies') ||
                route.path.startsWith('/proxy'),
            }"
            @click="closeSidebar"
          >
            Proxies
          </router-link>
        </nav>
      </aside>

      <main id="content">
        <router-view></router-view>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useDark } from '@vueuse/core'
import { Moon, Sunny } from '@element-plus/icons-vue'
import GitHubIcon from './assets/icons/github.svg?component'
import LogoIcon from './assets/icons/logo.svg?component'
import { useResponsive } from './composables/useResponsive'

const route = useRoute()
const isDark = useDark()
const { isMobile } = useResponsive()

const sidebarOpen = ref(false)

const toggleSidebar = () => {
  sidebarOpen.value = !sidebarOpen.value
}

const closeSidebar = () => {
  sidebarOpen.value = false
}

// Auto-close sidebar on route change
watch(
  () => route.path,
  () => {
    if (isMobile.value) {
      closeSidebar()
    }
  },
)
</script>

<style>
:root {
  --header-height: 50px;
  --sidebar-width: 200px;
  --header-bg: #ffffff;
  --header-border: #e4e7ed;
  --sidebar-bg: #ffffff;
  --text-primary: #303133;
  --text-secondary: #606266;
  --text-muted: #909399;
  --hover-bg: #efefef;
  --content-bg: #f9f9f9;
}

html.dark {
  --header-bg: #1e1e2e;
  --header-border: #3a3d5c;
  --sidebar-bg: #1e1e2e;
  --text-primary: #e5e7eb;
  --text-secondary: #b0b0b0;
  --text-muted: #888888;
  --hover-bg: #2a2a3e;
  --content-bg: #181825;
}

body {
  margin: 0;
  font-family:
    ui-sans-serif, -apple-system, system-ui, Segoe UI, Helvetica, Arial,
    sans-serif;
}

*,
:after,
:before {
  box-sizing: border-box;
  -webkit-tap-highlight-color: transparent;
}

html,
body {
  height: 100%;
  overflow: hidden;
}

#app {
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background-color: var(--content-bg);
}

/* Header */
.header {
  flex-shrink: 0;
  background: var(--header-bg);
  border-bottom: 1px solid var(--header-border);
  height: var(--header-height);
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 20px;
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
  width: 28px;
  height: 28px;
}

.divider {
  color: var(--header-border);
  font-size: 22px;
  font-weight: 200;
}

.brand-name {
  font-weight: 600;
  font-size: 18px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.badge {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
  background: var(--hover-bg);
  padding: 2px 8px;
  border-radius: 4px;
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
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  color: var(--text-secondary);
  transition: all 0.15s ease;
  text-decoration: none;
}

.github-link:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.github-icon {
  width: 18px;
  height: 18px;
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

/* Layout */
.layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: var(--sidebar-width);
  flex-shrink: 0;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--header-border);
  padding: 16px 12px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sidebar-link {
  display: block;
  text-decoration: none;
  font-size: 15px;
  color: var(--text-secondary);
  padding: 10px 12px;
  border-radius: 6px;
  transition: all 0.15s ease;
}

.sidebar-link:hover {
  color: var(--text-primary);
  background: var(--hover-bg);
}

.sidebar-link.active {
  color: var(--text-primary);
  background: var(--hover-bg);
  font-weight: 500;
}

/* Hamburger button (mobile only) */
.hamburger-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition: background 0.15s ease;
}

.hamburger-btn:hover {
  background: var(--hover-bg);
}

.hamburger-icon {
  font-size: 20px;
  line-height: 1;
  color: var(--text-primary);
}

/* Mobile overlay */
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 99;
}

/* Content */
#content {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 40px;
}

#content > * {
  max-width: 1024px;
  margin: 0 auto;
}

/* Common page styles */
.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary, var(--text-primary));
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-muted, var(--text-muted));
  margin: 8px 0 0;
}

/* Element Plus global overrides */
.el-button {
  font-weight: 500;
}

.el-tag {
  font-weight: 500;
}

.el-switch {
  --el-switch-on-color: #606266;
  --el-switch-off-color: #dcdfe6;
}

html.dark .el-switch {
  --el-switch-on-color: #b0b0b0;
  --el-switch-off-color: #3a3d5c;
}

.el-form-item {
  margin-bottom: 16px;
}

.el-loading-mask {
  border-radius: 8px;
}

/* Select overrides */
.el-select__wrapper {
  border-radius: 8px !important;
  box-shadow: 0 0 0 1px var(--color-border-light, #e4e7ed) inset !important;
  transition: all 0.15s ease;
}

.el-select__wrapper:hover {
  box-shadow: 0 0 0 1px var(--color-border, #dcdfe6) inset !important;
}

.el-select__wrapper.is-focused {
  box-shadow: 0 0 0 1px var(--color-border, #dcdfe6) inset !important;
}

.el-select-dropdown {
  border-radius: 12px !important;
  border: 1px solid var(--color-border-light, #e4e7ed) !important;
  box-shadow:
    0 10px 25px -5px rgba(0, 0, 0, 0.1),
    0 8px 10px -6px rgba(0, 0, 0, 0.1) !important;
  padding: 4px !important;
}

.el-select-dropdown__item {
  border-radius: 6px;
  margin: 2px 0;
  transition: background 0.15s ease;
}

.el-select-dropdown__item.is-selected {
  color: var(--color-text-primary, var(--text-primary));
  font-weight: 500;
}

/* Input overrides */
.el-input__wrapper {
  border-radius: 8px !important;
  box-shadow: 0 0 0 1px var(--color-border-light, #e4e7ed) inset !important;
  transition: all 0.15s ease;
}

.el-input__wrapper:hover {
  box-shadow: 0 0 0 1px var(--color-border, #dcdfe6) inset !important;
}

.el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px var(--color-border, #dcdfe6) inset !important;
}

/* Card overrides */
.el-card {
  border-radius: 12px;
  border: 1px solid var(--color-border-light, #e4e7ed);
  transition: all 0.2s ease;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #d1d1d1;
  border-radius: 3px;
}

/* Mobile */
@media (max-width: 767px) {
  .header-content {
    padding: 0 16px;
  }

  .sidebar {
    position: fixed;
    top: var(--header-height);
    left: 0;
    bottom: 0;
    z-index: 100;
    background: var(--sidebar-bg);
    transform: translateX(-100%);
    transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    border-right: 1px solid var(--header-border);
  }

  .sidebar.mobile-open {
    transform: translateX(0);
  }

  #content {
    width: 100%;
    padding: 20px;
  }
}
</style>
