<template>
  <div id="app">
    <header class="grid-content header-color">
      <div class="header-content">
        <div class="brand">
          <a href="#">{{ t("main.title") }}</a>
        </div>
        <div class="right-ability">
          <div class="lang-switch">
            <el-dropdown>
              <img src="./assets/lang.svg" alt="lang">
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :disabled="locale == 'en' ? true : false"
                    @click="switchLanguage('en')">English</el-dropdown-item>
                  <el-dropdown-item :disabled="locale == 'zh' ? true : false"
                    @click="switchLanguage('zh')">简体中文</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
          <div class="dark-switch">
            <el-switch v-model="darkmodeSwitch" inline-prompt :active-text="t('main.dark.Dark')"
              :inactive-text="t('main.dark.Light')" @change="toggleDark" style="
              --el-switch-on-color: #444452;
              --el-switch-off-color: #589ef8;
            " />
          </div>
        </div>
      </div>
    </header>
    <section>
      <el-row>
        <el-col id="side-nav" :xs="24" :md="4">
          <el-menu default-active="/" mode="vertical" theme="light" router="false" @select="handleSelect">
            <el-menu-item index="/">{{ t("main.Overview") }}</el-menu-item>
            <el-sub-menu index="/proxies">
              <template #title>
                <span>{{ t("main.Proxies.title") }}</span>
              </template>
              <el-menu-item index="/proxies/tcp">TCP</el-menu-item>
              <el-menu-item index="/proxies/udp">UDP</el-menu-item>
              <el-menu-item index="/proxies/http">HTTP</el-menu-item>
              <el-menu-item index="/proxies/https">HTTPS</el-menu-item>
              <el-menu-item index="/proxies/tcpmux">TCPMUX</el-menu-item>
              <el-menu-item index="/proxies/stcp">STCP</el-menu-item>
              <el-menu-item index="/proxies/sudp">SUDP</el-menu-item>
            </el-sub-menu>
            <el-menu-item index="">{{ t("main.Help") }}</el-menu-item>
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
import { useI18n } from 'vue-i18n';
const { t, locale } = useI18n();

const isDark = useDark()
const darkmodeSwitch = ref(isDark)
const toggleDark = useToggle(isDark)
const switchLanguage = (lang: string) => {
  locale.value = lang;
  localStorage.setItem('i18n', lang);
}
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

.right-ability {
  display: flex;
  justify-content: flex-end;
  flex-grow: 1;
  padding-right: 40px;
}

.lang-switch {
  width: 30px;
  margin-right: 10px;
}

.lang-switch img {
  width: 100%;
  height: auto;
}
</style>
