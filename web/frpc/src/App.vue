<template>
  <div id="app">
    <header class="header">
      <div class="header-content">
        <div class="brand-section">
          <button v-if="isMobile" class="hamburger-btn" @click="toggleSidebar" aria-label="Toggle menu">
            <span class="hamburger-icon">&#9776;</span>
          </button>
          <div class="logo-wrapper">
            <LogoIcon class="logo-icon" />
          </div>
          <span class="divider">/</span>
          <span class="brand-name">frp</span>
          <span class="badge">Client</span>
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

      <aside class="sidebar" :class="{ 'mobile-open': isMobile && sidebarOpen }">
        <nav class="sidebar-nav">
          <router-link
            to="/proxies"
            class="sidebar-link"
            :class="{ active: route.path.startsWith('/proxies') }"
            @click="closeSidebar"
          >
            Proxies
          </router-link>
          <router-link
            to="/visitors"
            class="sidebar-link"
            :class="{ active: route.path.startsWith('/visitors') }"
            @click="closeSidebar"
          >
            Visitors
          </router-link>
          <router-link
            to="/config"
            class="sidebar-link"
            :class="{ active: route.path === '/config' }"
            @click="closeSidebar"
          >
            Config
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
watch(() => route.path, () => {
  if (isMobile.value) {
    closeSidebar()
  }
})
</script>

<style lang="scss">
body {
  margin: 0;
  font-family: ui-sans-serif, -apple-system, system-ui, Segoe UI, Helvetica,
    Arial, sans-serif;
}

*,
:after,
:before {
  box-sizing: border-box;
  -webkit-tap-highlight-color: transparent;
}

html, body {
  height: 100%;
  overflow: hidden;
}

#app {
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background-color: $color-bg-secondary;
}

// Header
.header {
  flex-shrink: 0;
  background: $color-bg-primary;
  border-bottom: 1px solid $color-border-light;
  height: $header-height;
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 100%;
  padding: 0 $spacing-xl;
}

.brand-section {
  display: flex;
  align-items: center;
  gap: $spacing-md;
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
  color: $color-border;
  font-size: 22px;
  font-weight: 200;
}

.brand-name {
  font-weight: $font-weight-semibold;
  font-size: $font-size-xl;
  color: $color-text-primary;
  letter-spacing: -0.5px;
}

.badge {
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  color: $color-text-muted;
  background: $color-bg-muted;
  padding: 2px 8px;
  border-radius: 4px;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.github-link {
  @include flex-center;
  width: 28px;
  height: 28px;
  border-radius: $radius-sm;
  color: $color-text-secondary;
  transition: all $transition-fast;

  &:hover {
    background: $color-bg-hover;
    color: $color-text-primary;
  }
}

.github-icon {
  width: 18px;
  height: 18px;
}

.theme-switch {
  --el-switch-on-color: #2c2c3a;
  --el-switch-off-color: #f2f2f2;
  --el-switch-border-color: var(--color-border-light);
}

html.dark .theme-switch {
  --el-switch-off-color: #333;
}

.theme-switch .el-switch__core .el-switch__inner .el-icon {
  color: #909399 !important;
}

// Layout
.layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.sidebar {
  width: $sidebar-width;
  flex-shrink: 0;
  border-right: 1px solid $color-border-light;
  padding: $spacing-lg $spacing-md;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.sidebar-nav {
  @include flex-column;
  gap: 2px;
}

.sidebar-link {
  display: block;
  text-decoration: none;
  font-size: $font-size-lg;
  color: $color-text-secondary;
  padding: 10px $spacing-md;
  border-radius: $radius-sm;
  transition: all $transition-fast;

  &:hover {
    color: $color-text-primary;
    background: $color-bg-hover;
  }

  &.active {
    color: $color-text-primary;
    background: $color-bg-hover;
    font-weight: $font-weight-medium;
  }
}

// Hamburger button (mobile only)
.hamburger-btn {
  @include flex-center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: $radius-sm;
  background: transparent;
  cursor: pointer;
  padding: 0;
  transition: background $transition-fast;

  &:hover {
    background: $color-bg-hover;
  }
}

.hamburger-icon {
  font-size: 20px;
  line-height: 1;
  color: $color-text-primary;
}

// Mobile overlay
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 99;
}

#content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  background: $color-bg-primary;
}

// Common page styles
.page-title {
  font-size: $font-size-xl + 2px;
  font-weight: $font-weight-semibold;
  color: $color-text-primary;
  margin: 0;
}

.page-subtitle {
  font-size: $font-size-md;
  color: $color-text-muted;
  margin: $spacing-sm 0 0;
}

.icon-btn {
  @include flex-center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: $radius-sm;
  background: transparent;
  color: $color-text-muted;
  cursor: pointer;
  transition: all $transition-fast;

  &:hover {
    background: $color-bg-hover;
    color: $color-text-primary;
  }
}

.search-input {
  width: 200px;

  .el-input__wrapper {
    border-radius: 10px;
    background: $color-bg-tertiary;
    box-shadow: 0 0 0 1px $color-border inset;

    &.is-focus {
      box-shadow: 0 0 0 1px $color-text-light inset;
    }
  }

  .el-input__inner {
    color: $color-text-primary;
  }

  .el-input__prefix {
    color: $color-text-muted;
  }

  @include mobile {
    flex: 1;
    width: auto;
  }
}

// Element Plus global overrides
.el-button {
  font-weight: $font-weight-medium;
}

.el-tag {
  font-weight: $font-weight-medium;
}

.el-switch {
  --el-switch-on-color: #606266;
  --el-switch-off-color: #dcdfe6;
}

html.dark .el-switch {
  --el-switch-on-color: #b0b0b0;
  --el-switch-off-color: #404040;
}

.el-radio {
  --el-radio-text-color: var(--color-text-primary) !important;
  --el-radio-input-border-color-hover: #606266 !important;
  --el-color-primary: #606266 !important;
}

.el-form-item {
  margin-bottom: 16px;
}

.el-loading-mask {
  border-radius: $radius-md;
}

// Select overrides
.el-select__wrapper {
  border-radius: $radius-md !important;
  box-shadow: 0 0 0 1px $color-border-light inset !important;
  transition: all $transition-fast;

  &:hover {
    box-shadow: 0 0 0 1px $color-border inset !important;
  }

  &.is-focused {
    box-shadow: 0 0 0 1px $color-border inset !important;
  }
}

.el-select-dropdown {
  border-radius: 12px !important;
  border: 1px solid $color-border-light !important;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1),
              0 8px 10px -6px rgba(0, 0, 0, 0.1) !important;
  padding: 4px !important;
}

.el-select-dropdown__item {
  border-radius: $radius-sm;
  margin: 2px 0;
  transition: background $transition-fast;

  &.is-selected {
    color: $color-text-primary;
    font-weight: $font-weight-medium;
  }
}

// Input overrides
.el-input__wrapper {
  border-radius: $radius-md !important;
  box-shadow: 0 0 0 1px $color-border-light inset !important;
  transition: all $transition-fast;

  &:hover {
    box-shadow: 0 0 0 1px $color-border inset !important;
  }

  &.is-focus {
    box-shadow: 0 0 0 1px $color-border inset !important;
  }
}

// Status pill (shared)
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  padding: 3px 10px;
  border-radius: 10px;
  text-transform: capitalize;

  &.running {
    background: rgba(103, 194, 58, 0.1);
    color: #67c23a;
  }

  &.error {
    background: rgba(245, 108, 108, 0.1);
    color: #f56c6c;
  }

  &.waiting {
    background: rgba(230, 162, 60, 0.1);
    color: #e6a23c;
  }

  &.disabled {
    background: $color-bg-muted;
    color: $color-text-light;
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }
}

// Mobile
@include mobile {
  .header-content {
    padding: 0 $spacing-lg;
  }

  .sidebar {
    position: fixed;
    top: $header-height;
    left: 0;
    bottom: 0;
    z-index: 100;
    background: $color-bg-primary;
    transform: translateX(-100%);
    transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
    border-right: 1px solid $color-border-light;

    &.mobile-open {
      transform: translateX(0);
    }
  }

  .sidebar-nav {
    flex-direction: column;
    gap: 2px;
  }

  #content {
    width: 100%;
  }

  // Select dropdown overflow prevention
  .el-select-dropdown {
    max-width: calc(100vw - 32px);
  }
}
</style>
