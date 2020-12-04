<template>
  <div></div>
</template>

<script>
export default {
  created() {
    this.$router.push(`/${process.env.VUE_APP_TYPE}`)
  }
}
</script>
