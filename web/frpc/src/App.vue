<template>
  <div id="app">
    <header class="header">
      <div class="header-top">
        <div class="brand">
          <a href="#" @click.prevent="router.push('/')">frpc</a>
        </div>
        <div class="header-actions">
          <a
            class="github-link"
            href="https://github.com/fatedier/frp"
            target="_blank"
            aria-label="GitHub"
          >
            <GitHubIcon class="github-icon" />
          </a>
          <el-switch
            v-model="darkmodeSwitch"
            inline-prompt
            :active-icon="Moon"
            :inactive-icon="Sunny"
            @change="toggleDark"
            class="theme-switch"
          />
        </div>
      </div>
      <nav class="header-nav">
        <el-menu
          :default-active="currentRoute"
          mode="horizontal"
          :ellipsis="false"
          @select="handleSelect"
          class="nav-menu"
        >
          <el-menu-item index="/">Overview</el-menu-item>
          <el-menu-item index="/configure">Configure</el-menu-item>
        </el-menu>
      </nav>
    </header>
    <main id="content">
      <router-view></router-view>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useDark, useToggle } from '@vueuse/core'
import { Moon, Sunny } from '@element-plus/icons-vue'
import GitHubIcon from './assets/icons/github.svg?component'

const router = useRouter()
const route = useRoute()
const isDark = useDark()
const darkmodeSwitch = ref(isDark)
const toggleDark = useToggle(isDark)

const currentRoute = computed(() => {
  return route.path
})

const handleSelect = (key: string) => {
  router.push(key)
}
</script>

<style>
body {
  margin: 0;
  font-family:
    -apple-system,
    BlinkMacSystemFont,
    Helvetica Neue,
    sans-serif;
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f2f2f2;
}

html.dark #app {
  background: #1a1a2e;
}

.header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: #fff;
}

html.dark .header {
  background: #1e1e2d;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 32px;
}

.brand a {
  color: #303133;
  font-size: 20px;
  font-weight: 700;
  text-decoration: none;
  letter-spacing: -0.5px;
}

html.dark .brand a {
  color: #e5e7eb;
}

.brand a:hover {
  color: #409eff;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.github-link {
  display: flex;
  align-items: center;
  padding: 6px;
  border-radius: 6px;
  transition: all 0.2s;
}

.github-link:hover {
  background: #f2f3f5;
}

html.dark .github-link:hover {
  background: #2a2a3c;
}

.github-icon {
  width: 20px;
  height: 20px;
  color: #606266;
  transition: color 0.2s;
}

.github-link:hover .github-icon {
  color: #303133;
}

html.dark .github-icon {
  color: #a0a3ad;
}

html.dark .github-link:hover .github-icon {
  color: #e5e7eb;
}

.theme-switch {
  --el-switch-on-color: #2c2c3a;
  --el-switch-off-color: #f2f2f2;
  --el-switch-border-color: #dcdfe6;
}

.theme-switch .el-switch__core .el-switch__inner .el-icon {
  color: #909399 !important;
}

.header-nav {
  position: relative;
  padding: 0 32px;
  border-bottom: 1px solid #e4e7ed;
}

html.dark .header-nav {
  border-bottom-color: #3a3d5c;
}

.nav-menu {
  background: transparent !important;
  border-bottom: none !important;
  height: 46px;
}

.nav-menu .el-menu-item,
.nav-menu .el-sub-menu__title {
  position: relative;
  height: 32px !important;
  line-height: 32px !important;
  border-bottom: none !important;
  border-radius: 6px !important;
  color: #666 !important;
  font-weight: 400;
  font-size: 14px;
  padding: 0 12px !important;
  margin: 7px 0;
  transition:
    background 0.15s ease,
    color 0.15s ease;
}

.nav-menu > .el-menu-item,
.nav-menu > .el-sub-menu {
  margin-right: 4px;
}

.nav-menu > .el-sub-menu {
  padding: 0 !important;
}

html.dark .nav-menu .el-menu-item,
html.dark .nav-menu .el-sub-menu__title {
  color: #888 !important;
}

.nav-menu .el-menu-item:hover,
.nav-menu .el-sub-menu__title:hover {
  background: #f2f2f2 !important;
  color: #171717 !important;
}

html.dark .nav-menu .el-menu-item:hover,
html.dark .nav-menu .el-sub-menu__title:hover {
  background: #2a2a3c !important;
  color: #e5e7eb !important;
}

.nav-menu .el-menu-item.is-active {
  background: transparent !important;
  color: #171717 !important;
  font-weight: 500;
}

.nav-menu .el-menu-item.is-active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -3px;
  height: 2px;
  background: #171717;
  border-radius: 1px;
}

.nav-menu .el-menu-item.is-active:hover {
  background: #f2f2f2 !important;
}

html.dark .nav-menu .el-menu-item.is-active {
  background: transparent !important;
  color: #e5e7eb !important;
  font-weight: 500;
}

html.dark .nav-menu .el-menu-item.is-active::after {
  background: #e5e7eb;
}

html.dark .nav-menu .el-menu-item.is-active:hover {
  background: #2a2a3c !important;
}

#content {
  flex: 1;
  padding: 24px 40px;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

@media (max-width: 768px) {
  .header-top {
    padding: 0 16px;
  }

  .header-nav {
    padding: 0 16px;
  }

  #content {
    padding: 16px;
  }

  .brand a {
    font-size: 18px;
  }
}
</style>