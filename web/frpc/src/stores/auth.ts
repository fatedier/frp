import { checkAuthRequest } from '../api/auth'
import { authenticated, checked, resetAuthState, setAuthenticated } from './authState'

export { authenticated, checked, resetAuthState, setAuthenticated }

// checkAuthState probes the server once and caches the result. It is called
// by the router guard on the first navigation; after a login or a mid-session
// 401 the cache is updated through setAuthenticated/resetAuthState.
export async function checkAuthState(): Promise<boolean> {
  if (checked.value) return authenticated.value
  try {
    await checkAuthRequest()
    authenticated.value = true
  } catch {
    authenticated.value = false
  }
  checked.value = true
  return authenticated.value
}
