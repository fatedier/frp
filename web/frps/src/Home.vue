<template>
  <div id="container">
    <div id="wrap">
      <div id="top">
        <div id="logo">
          <!--<img class="logo"-->
          <!--     src="img/logo.png"-->
          <!--/>-->
          <!--如果需要自定义logo图片,使用上面这段代码,并删除下面这段代码,图片位置应放在img下的logo.png-->
          <router-link to="/overview">
            <img
              class="logo"
              src="data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBzdGFuZGFsb25lPSJubyI/PjwhRE9DVFlQRSBzdmcgUFVCTElDICItLy9XM0MvL0RURCBTVkcgMS4xLy9FTiIgImh0dHA6Ly93d3cudzMub3JnL0dyYXBoaWNzL1NWRy8xLjEvRFREL3N2ZzExLmR0ZCI+PHN2ZyB0PSIxNjgxODk0Mjk2MzM3IiBjbGFzcz0iaWNvbiIgdmlld0JveD0iMCAwIDEwMjQgMTAyNCIgdmVyc2lvbj0iMS4xIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHAtaWQ9IjI2NjEiIHhtbG5zOnhsaW5rPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5L3hsaW5rIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjIwMCI+PHBhdGggZD0iTTc4NS4wNjY2NjcgODUzLjMzMzMzM0gyMzguOTMzMzMzYy0xMzMuMTIgMC0yMzguOTMzMzMzLTEwNS44MTMzMzMtMjM4LjkzMzMzMy0yMzguOTMzMzMzIDAtMTA1LjgxMzMzMyA3MS42OC0xOTcuOTczMzMzIDE3MC42NjY2NjctMjI4LjY5MzMzMyA2LjgyNjY2Ny03MS42OCA3MS42OC0xMjkuNzA2NjY3IDE0My4zNi0xMjkuNzA2NjY3aDYuODI2NjY2YzIwLjQ4IDAgNDAuOTYgMy40MTMzMzMgNjEuNDQgMTMuNjUzMzMzQzQyNi42NjY2NjcgMjA4LjIxMzMzMyA0OTguMzQ2NjY3IDE3MC42NjY2NjcgNTczLjQ0IDE3MC42NjY2NjdoNi44MjY2NjdjMTE5LjQ2NjY2NyAwIDIyMS44NjY2NjcgODguNzQ2NjY3IDIzNS41MiAyMDguMjEzMzMzIDExNi4wNTMzMzMgMTcuMDY2NjY3IDIwOC4yMTMzMzMgMTE2LjA1MzMzMyAyMDguMjEzMzMzIDIzNS41MiAwIDEzMy4xMi0xMDUuODEzMzMzIDIzOC45MzMzMzMtMjM4LjkzMzMzMyAyMzguOTMzMzMzek0zMTcuNDQgMjkwLjEzMzMzM0MyNTYgMjkwLjEzMzMzMyAyMDQuOCAzMzcuOTIgMjA0LjggMzk5LjM2YzAgNi44MjY2NjctNi44MjY2NjcgMTMuNjUzMzMzLTEzLjY1MzMzMyAxNy4wNjY2NjctOTIuMTYgMjAuNDgtMTU3LjAxMzMzMyAxMDIuNC0xNTcuMDEzMzM0IDE5Ny45NzMzMzMgMCAxMTIuNjQgOTIuMTYgMjA0LjggMjA0LjggMjA0LjhoNTQ2LjEzMzMzNGMxMTIuNjQgMCAyMDQuOC05Mi4xNiAyMDQuOC0yMDQuOCAwLTEwNS44MTMzMzMtODEuOTItMTk3Ljk3MzMzMy0xOTEuMTQ2NjY3LTIwNC44LTYuODI2NjY3IDAtMTMuNjUzMzMzLTYuODI2NjY3LTE3LjA2NjY2Ny0xNy4wNjY2NjdDNzc4LjI0IDI4Ni43MiA2ODYuMDggMjA0LjggNTgwLjI2NjY2NyAyMDQuOGgtMy40MTMzMzRsLTMuNDEzMzMzLTE3LjA2NjY2N1YyMDQuOGMtNjguMjY2NjY3IDAtMTI5LjcwNjY2NyAzNC4xMzMzMzMtMTY3LjI1MzMzMyA5NS41NzMzMzMtNi44MjY2NjcgNi44MjY2NjctMTcuMDY2NjY3IDEwLjI0LTIzLjg5MzMzNCA2LjgyNjY2Ny0xNy4wNjY2NjctMTAuMjQtMzcuNTQ2NjY3LTE3LjA2NjY2Ny01OC4wMjY2NjYtMTcuMDY2NjY3aC0zLjQxMzMzNC0zLjQxMzMzM3oiIGZpbGw9IiNmZmZmZmYiIHAtaWQ9IjI2NjIiPjwvcGF0aD48L3N2Zz4="
            />
          </router-link>
          <p></p>
          <div class="headline">
            <h1>Welcome</h1>
          </div>
          <div class="" id="kg-btn">
            <input class="tgl tgl-flip" id="qieh" type="checkbox" />
            <label
              class="tgl-btn"
              data-tg-off="PROXY"
              data-tg-on="DIRECT"
              for="qieh"
              @click="toggleProxy"
            ></label>
          </div>
        </div>
      </div>
      <div id="main">
        <div class="app animated fadeInLeft" id="app">
          <ul>
            <li v-for="p in proxies">
              <a :href="isProxy ? p.proxyUrl : p.directUrl" target="_blank">
                <img class="shake" width="128" height="128" :src="p.image" />
                <strong>{{ p.name }}</strong></a
              >
            </li>
          </ul>
        </div>
      </div>
    </div>
    <div id="footer">
      <div class="footer-contents">
        <div class="links">
          <div class="line">
            <a href="/">FRP Home</a>&nbsp;&nbsp;&nbsp;
            <span class="footer-link-separator"></span>&nbsp;&nbsp;&nbsp;
            <span class="copyright">@</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
class Item {
  name: string
  image: string
  directUrl: string
  proxyUrl: string

  /**
   *
   */
  constructor(proxy: any, host: string) {
    this.name = proxy.name
    this.image = proxy.conf.group
    this.directUrl = proxy.conf.group_key
    this.proxyUrl = `${host}:${proxy.conf.remote_port}`
  }
}

let proxies = ref<Item[]>([])
let isProxy = ref<Boolean>(true)
const protocal = window.location.protocol
const host = window.location.hostname

const fetchData = () => {
  fetch('../api/proxy/tcp', { credentials: 'include' })
    .then((res) => {
      return res.json()
    })
    .then((json) => {
      for (let proxy of json.proxies) {
        if (proxy != null && proxy.conf.group != '')
          proxies.value.push(new Item(proxy, `${protocal}//${host}`))
      }
    })
}
fetchData()

const toggleProxy = () => {
  isProxy.value = !isProxy.value
}
</script>

<style scoped>
@import url('assets/style.css');
</style>
