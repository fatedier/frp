import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ProxyStatus, ProxyDefinition } from '../types'
import {
  getStatus,
  listStoreProxies,
  getStoreProxy,
  createStoreProxy,
  updateStoreProxy,
  deleteStoreProxy,
} from '../api/frpc'

export const useProxyStore = defineStore('proxy', () => {
  const proxies = ref<ProxyStatus[]>([])
  const storeProxies = ref<ProxyDefinition[]>([])
  const storeEnabled = ref(false)
  const storeChecked = ref(false)
  const loading = ref(false)
  const storeLoading = ref(false)
  const error = ref<string | null>(null)

  const fetchStatus = async () => {
    loading.value = true
    error.value = null
    try {
      const json = await getStatus()
      const list: ProxyStatus[] = []
      for (const key in json) {
        for (const ps of json[key]) {
          list.push(ps)
        }
      }
      proxies.value = list
    } catch (err: any) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchStoreProxies = async () => {
    storeLoading.value = true
    try {
      const res = await listStoreProxies()
      storeProxies.value = res.proxies || []
      storeEnabled.value = true
      storeChecked.value = true
    } catch (err: any) {
      if (err?.status === 404) {
        storeEnabled.value = false
      }
      storeChecked.value = true
    } finally {
      storeLoading.value = false
    }
  }

  const checkStoreEnabled = async () => {
    if (storeChecked.value) return storeEnabled.value
    await fetchStoreProxies()
    return storeEnabled.value
  }

  const createProxy = async (data: ProxyDefinition) => {
    await createStoreProxy(data)
    await fetchStoreProxies()
  }

  const updateProxy = async (name: string, data: ProxyDefinition) => {
    await updateStoreProxy(name, data)
    await fetchStoreProxies()
  }

  const deleteProxy = async (name: string) => {
    await deleteStoreProxy(name)
    await fetchStoreProxies()
  }

  const toggleProxy = async (name: string, enabled: boolean) => {
    const def = await getStoreProxy(name)
    const block = (def as any)[def.type]
    if (block) {
      block.enabled = enabled
    }
    await updateStoreProxy(name, def)
    await fetchStatus()
    await fetchStoreProxies()
  }

  const storeProxyWithStatus = (def: ProxyDefinition): ProxyStatus => {
    const block = (def as any)[def.type]
    const enabled = block?.enabled !== false

    const localIP = block?.localIP || '127.0.0.1'
    const localPort = block?.localPort
    const local_addr = localPort != null ? `${localIP}:${localPort}` : ''
    const remotePort = block?.remotePort
    const remote_addr = remotePort != null ? `:${remotePort}` : ''
    const plugin = block?.plugin?.type || ''

    const status = proxies.value.find((p) => p.name === def.name)
    return {
      name: def.name,
      type: def.type,
      status: !enabled ? 'disabled' : (status?.status || 'waiting'),
      err: status?.err || '',
      local_addr: status?.local_addr || local_addr,
      remote_addr: status?.remote_addr || remote_addr,
      plugin: status?.plugin || plugin,
      source: 'store',
    }
  }

  return {
    proxies,
    storeProxies,
    storeEnabled,
    storeChecked,
    loading,
    storeLoading,
    error,
    fetchStatus,
    fetchStoreProxies,
    checkStoreEnabled,
    createProxy,
    updateProxy,
    deleteProxy,
    toggleProxy,
    storeProxyWithStatus,
  }
})
