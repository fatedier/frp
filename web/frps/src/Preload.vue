<template>
  <div v-if="isHome">
    <router-view />
  </div>
  <div v-else>
    <App />
  </div>
</template>

<script lang="ts">
import App from './App.vue'

export default {
  components: {
    App,
  },
  data() {
    return {
      isHome: false,
    }
  },
  mounted() {
    this.isHome = document.location.hash == '#/'
  },
  watch: {
    $route(to, from) {
      this.isHome = to.path == '/'
    },
  },
}
</script>
