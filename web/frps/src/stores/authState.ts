import { ref } from 'vue'

// Leaf auth state module: it must not import anything that depends on the
// http client, so that http.ts can reset the state on a 401 without creating
// an import cycle.
export const authenticated = ref(false)
export const checked = ref(false)

export function setAuthenticated(value: boolean) {
  authenticated.value = value
  checked.value = true
}

export function resetAuthState() {
  authenticated.value = false
  checked.value = true
}
