import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import svgLoader from 'vite-svg-loader'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import ElementPlus from 'unplugin-element-plus/vite'

// https://vitejs.dev/config/
export default defineConfig({
  base: '',
  plugins: [
    vue(),
    svgLoader(),
    ElementPlus({}),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@shared': fileURLToPath(new URL('../shared', import.meta.url)),
    },
    dedupe: ['vue', 'element-plus', '@element-plus/icons-vue'],
    modules: [
      fileURLToPath(new URL('../node_modules', import.meta.url)),
      'node_modules',
    ],
  },
  css: {
    preprocessorOptions: {
      scss: {
        api: 'modern',
        additionalData: `@use "@shared/css/_index.scss" as *;`,
      },
    },
  },
  build: {
    assetsDir: '',
    chunkSizeWarningLimit: 1000,
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
      },
    },
  },
  server: {
    allowedHosts: process.env.ALLOWED_HOSTS
      ? process.env.ALLOWED_HOSTS.split(',')
      : [],
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://127.0.0.1:7400',
        changeOrigin: true,
      },
    },
  },
})
