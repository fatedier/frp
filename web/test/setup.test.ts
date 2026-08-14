import { describe, expect, it, vi } from 'vitest'

const capturedFetch = globalThis.fetch

describe('unit-test environment', () => {
  it('blocks real network requests before test modules are evaluated', async () => {
    await expect(capturedFetch('https://example.invalid/')).rejects.toThrow(
      'Real network requests are disabled in unit tests',
    )
  })

  it('allows a test to replace fetch explicitly', async () => {
    const replacement = vi.fn(async () => new Response('ok'))
    vi.stubGlobal('fetch', replacement)

    await expect(fetch('/test-endpoint')).resolves.toBeInstanceOf(Response)
    expect(replacement).toHaveBeenCalledWith('/test-endpoint')
  })

  it('restores the network guard after a test replacement', async () => {
    await expect(fetch('/must-remain-isolated')).rejects.toThrow(
      'Real network requests are disabled in unit tests',
    )
  })
})
