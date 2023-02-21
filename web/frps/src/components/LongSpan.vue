<template>
  <el-tooltip :content="content" placement="top">
    <span v-show="content.length > length"
      >{{ content.slice(0, length) }}...</span
    >
  </el-tooltip>
  <span v-show="content.length < 30">{{ content }}</span>
</template>

<script setup lang="ts">
defineProps<{
  content: string
  length: number
}>()
</script>
