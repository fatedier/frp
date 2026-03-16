import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getConfig, putConfig, reloadConfig } from '../api/frpc'

export const useClientStore = defineStore('client', () => {
  const config = ref('')
  const loading = ref(false)

  const fetchConfig = async () => {
    loading.value = true
    try {
      config.value = await getConfig()
    } finally {
      loading.value = false
    }
  }

  const saveConfig = async (text: string) => {
    await putConfig(text)
    config.value = text
  }

  const reload = async () => {
    await reloadConfig()
  }

  return { config, loading, fetchConfig, saveConfig, reload }
})
