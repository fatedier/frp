import { enableAutoUnmount } from '@vue/test-utils'
import { afterAll, afterEach, vi } from 'vitest'

const originalFetch = globalThis.fetch
const blockedFetch = vi.fn(async () => {
  throw new Error('Real network requests are disabled in unit tests')
})

globalThis.fetch = blockedFetch as typeof fetch

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
  globalThis.fetch = blockedFetch as typeof fetch
  document.body.innerHTML = ''
})

// Vitest runs afterEach hooks in reverse registration order, so wrappers are
// unmounted before the cleanup hook above clears the DOM.
enableAutoUnmount(afterEach)

afterAll(() => {
  globalThis.fetch = originalFetch
})
