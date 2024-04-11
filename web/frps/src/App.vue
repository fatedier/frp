<template>
  <div id="app">
    <header class="grid-content header-color">
      <div class="header-content">
        <div class="brand">
          <a href="#">frp</a>
        </div>
        <div class="dark-switch">
          <el-switch
            v-model="darkmodeSwitch"
            inline-prompt
            active-text="Dark"
            inactive-text="Light"
            @change="toggleDark"
            style="
              --el-switch-on-color: #444452;
              --el-switch-off-color: #589ef8;
            "
          />
        </div>
      </div>
    </header>
    <section>
      <el-row>
        <el-col id="side-nav" :xs="24" :md="4">
          <el-menu
            default-active="/"
            mode="vertical"
            theme="light"
            router="false"
            @select="handleSelect"
          >
            <el-menu-item index="/">Overview</el-menu-item>
            <el-sub-menu index="/proxies">
              <template #title>
                <span>Proxies</span>
              </template>
              <el-menu-item index="/proxies/tcp">TCP</el-menu-item>
              <el-menu-item index="/proxies/udp">UDP</el-menu-item>
              <el-menu-item index="/proxies/http">HTTP</el-menu-item>
              <el-menu-item index="/proxies/https">HTTPS</el-menu-item>
              <el-menu-item index="/proxies/tcpmux">TCPMUX</el-menu-item>
              <el-menu-item index="/proxies/stcp">STCP</el-menu-item>
              <el-menu-item index="/proxies/sudp">SUDP</el-menu-item>
            </el-sub-menu>
            <el-menu-item index="">Help</el-menu-item>
          </el-menu>
        </el-col>

        <el-col :xs="24" :md="20">
          <div id="content">
            <router-view></router-view>
          </div>
        </el-col>
      </el-row>
    </section>
    <footer></footer>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useDark, useToggle } from '@vueuse/core'

const isDark = useDark()
const darkmodeSwitch = ref(isDark)
const toggleDark = useToggle(isDark)

const handleSelect = (key: string) => {
  if (key == '') {
    window.open('https://github.com/fatedier/frp')
  }
}
</script>

<style>
body {
  margin: 0px;
  font-family: -apple-system, BlinkMacSystemFont, Helvetica Neue, sans-serif;
}

header {
  width: 100%;
  height: 60px;
}

.header-color {
  background: #58b7ff;
}

html.dark .header-color {
  background: #395c74;
}

.header-content {
  display: flex;
  align-items: center;
}

#content {
  margin-top: 20px;
  padding-right: 40px;
}

.brand {
  display: flex;
  justify-content: flex-start;
}

.brand a {
  color: #fff;
  background-color: transparent;
  margin-left: 20px;
  line-height: 25px;
  font-size: 25px;
  padding: 15px 15px;
  height: 30px;
  text-decoration: none;
}

.dark-switch {
  display: flex;
  justify-content: flex-end;
  flex-grow: 1;
  padding-right: 40px;
}
</style>
