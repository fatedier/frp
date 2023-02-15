<template>
  <div id="app">
    <header class="grid-content header-color">
      <el-row>
        <a class="brand" href="#">frp</a>
      </el-row>
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
const handleSelect = (key: string) => {
  if (key == '') {
    window.open('https://github.com/fatedier/frp')
  }
}
</script>

<style>
body {
  background-color: #fafafa;
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

#content {
  margin-top: 20px;
  padding-right: 40px;
}

.brand {
  color: #fff;
  background-color: transparent;
  margin-left: 20px;
  float: left;
  line-height: 25px;
  font-size: 25px;
  padding: 15px 15px;
  height: 30px;
  text-decoration: none;
}
</style>
