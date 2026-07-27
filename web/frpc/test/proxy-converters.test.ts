import { describe, expect, it } from 'vitest'
import {
  createDefaultProxyForm,
  createDefaultVisitorForm,
} from '../src/types/proxy-form'
import {
  formToStoreProxy,
  formToStoreVisitor,
  storeProxyToForm,
} from '../src/types/proxy-converters'

describe('proxy converters', () => {
  it('serializes only configured proxy fields into the typed store block', () => {
    const form = createDefaultProxyForm()
    Object.assign(form, {
      name: 'web',
      type: 'http',
      enabled: false,
      localIP: '10.0.0.8',
      localPort: 8080,
      useEncryption: true,
      customDomains: ['example.com', ''],
      locations: ['/api', ''],
      metadatas: [{ key: 'team', value: 'edge' }],
    })

    expect(formToStoreProxy(form)).toEqual({
      name: 'web',
      type: 'http',
      http: {
        enabled: false,
        localIP: '10.0.0.8',
        localPort: 8080,
        transport: { useEncryption: true },
        metadatas: { team: 'edge' },
        customDomains: ['example.com'],
        locations: ['/api'],
      },
    })
  })

  it('round-trips store values while applying form defaults', () => {
    const form = storeProxyToForm({
      name: 'tcp-proxy',
      type: 'tcp',
      tcp: {
        localPort: 9000,
        customDomains: 'unused-for-tcp',
        enabled: false,
        metadatas: { owner: 'platform' },
      },
    })

    expect(form).toMatchObject({
      name: 'tcp-proxy',
      type: 'tcp',
      enabled: false,
      localIP: '127.0.0.1',
      localPort: 9000,
      metadatas: [{ key: 'owner', value: 'platform' }],
      multiplexer: 'httpconnect',
    })
  })

  it.each(['stcp', 'xtcp'] as const)(
    'serializes virtual_net for %s visitors',
    (type) => {
      const form = createDefaultVisitorForm()
      Object.assign(form, {
        name: 'visitor',
        type,
        pluginType: 'virtual_net',
        pluginDestinationIP: ' 192.0.2.10 ',
        bindPort: 6000,
      })

      expect(formToStoreVisitor(form)[type]).toMatchObject({
        plugin: { type: 'virtual_net', destinationIP: '192.0.2.10' },
        bindPort: 6000,
      })
    },
  )

  it('does not serialize virtual_net for SUDP visitors', () => {
    const form = createDefaultVisitorForm()
    Object.assign(form, {
      name: 'visitor',
      type: 'sudp',
      pluginType: 'virtual_net',
      pluginDestinationIP: '192.0.2.10',
      bindPort: 6000,
    })

    expect(formToStoreVisitor(form).sudp).toEqual({ bindPort: 6000 })
  })
})
