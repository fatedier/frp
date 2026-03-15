import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { VisitorDefinition } from '../types'
import {
  listStoreVisitors,
  createStoreVisitor,
  updateStoreVisitor,
  deleteStoreVisitor,
} from '../api/frpc'

export const useVisitorStore = defineStore('visitor', () => {
  const storeVisitors = ref<VisitorDefinition[]>([])
  const storeEnabled = ref(false)
  const storeChecked = ref(false)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchStoreVisitors = async () => {
    loading.value = true
    try {
      const res = await listStoreVisitors()
      storeVisitors.value = res.visitors || []
      storeEnabled.value = true
      storeChecked.value = true
    } catch (err: any) {
      if (err?.status === 404) {
        storeEnabled.value = false
      }
      storeChecked.value = true
    } finally {
      loading.value = false
    }
  }

  const checkStoreEnabled = async () => {
    if (storeChecked.value) return storeEnabled.value
    await fetchStoreVisitors()
    return storeEnabled.value
  }

  const createVisitor = async (data: VisitorDefinition) => {
    await createStoreVisitor(data)
    await fetchStoreVisitors()
  }

  const updateVisitor = async (name: string, data: VisitorDefinition) => {
    await updateStoreVisitor(name, data)
    await fetchStoreVisitors()
  }

  const deleteVisitor = async (name: string) => {
    await deleteStoreVisitor(name)
    await fetchStoreVisitors()
  }

  return {
    storeVisitors,
    storeEnabled,
    storeChecked,
    loading,
    error,
    fetchStoreVisitors,
    checkStoreEnabled,
    createVisitor,
    updateVisitor,
    deleteVisitor,
  }
})
