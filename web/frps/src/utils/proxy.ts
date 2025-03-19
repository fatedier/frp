class BaseProxy {
  name: string
  type: string
  annotations: Map<string, string>
  encryption: boolean
  compression: boolean
  conns: number
  trafficIn: number
  trafficOut: number
  lastStartTime: string
  lastCloseTime: string
  status: string
  clientVersion: string
  addr: string
  port: number

  customDomains: string
  hostHeaderRewrite: string
  locations: string
  subdomain: string

  constructor(proxyStats: any) {
    this.name = proxyStats.name
    this.type = ''
    this.annotations = new Map<string, string>()
    if (proxyStats.conf?.annotations) {
      for (const key in proxyStats.conf.annotations) {
        this.annotations.set(key, proxyStats.conf.annotations[key])
      }
    }

    this.encryption = false
    this.compression = false
    this.encryption =
      proxyStats.conf?.transport?.useEncryption || this.encryption
    this.compression =
      proxyStats.conf?.transport?.useCompression || this.compression
    this.conns = proxyStats.curConns
    this.trafficIn = proxyStats.todayTrafficIn
    this.trafficOut = proxyStats.todayTrafficOut
    this.lastStartTime = proxyStats.lastStartTime
    this.lastCloseTime = proxyStats.lastCloseTime
    this.status = proxyStats.status
    this.clientVersion = proxyStats.clientVersion

    this.addr = ''
    this.port = 0
    this.customDomains = ''
    this.hostHeaderRewrite = ''
    this.locations = ''
    this.subdomain = ''
  }
}

class TCPProxy extends BaseProxy {
  constructor(proxyStats: any) {
    super(proxyStats)
    this.type = 'tcp'
    if (proxyStats.conf != null) {
      this.addr = ':' + proxyStats.conf.remotePort
      this.port = proxyStats.conf.remotePort
    } else {
      this.addr = ''
      this.port = 0
    }
  }
}

class UDPProxy extends BaseProxy {
  constructor(proxyStats: any) {
    super(proxyStats)
    this.type = 'udp'
    if (proxyStats.conf != null) {
      this.addr = ':' + proxyStats.conf.remotePort
      this.port = proxyStats.conf.remotePort
    } else {
      this.addr = ''
      this.port = 0
    }
  }
}

class HTTPProxy extends BaseProxy {
  constructor(proxyStats: any, port: number, subdomainHost: string) {
    super(proxyStats)
    this.type = 'http'
    this.port = port
    if (proxyStats.conf) {
      this.customDomains = proxyStats.conf.customDomains || this.customDomains
      this.hostHeaderRewrite = proxyStats.conf.hostHeaderRewrite
      this.locations = proxyStats.conf.locations
      if (proxyStats.conf.subdomain) {
        this.subdomain = `${proxyStats.conf.subdomain}.${subdomainHost}`
      }
    }
  }
}

class HTTPSProxy extends BaseProxy {
  constructor(proxyStats: any, port: number, subdomainHost: string) {
    super(proxyStats)
    this.type = 'https'
    this.port = port
    if (proxyStats.conf != null) {
      this.customDomains = proxyStats.conf.customDomains || this.customDomains
      if (proxyStats.conf.subdomain) {
        this.subdomain = `${proxyStats.conf.subdomain}.${subdomainHost}`
      }
    }
  }
}

class TCPMuxProxy extends BaseProxy {
  multiplexer: string
  routeByHTTPUser: string

  constructor(proxyStats: any, port: number, subdomainHost: string) {
    super(proxyStats)
    this.type = 'tcpmux'
    this.port = port
    this.multiplexer = ''
    this.routeByHTTPUser = ''

    if (proxyStats.conf) {
      this.customDomains = proxyStats.conf.customDomains || this.customDomains
      this.multiplexer = proxyStats.conf.multiplexer
      this.routeByHTTPUser = proxyStats.conf.routeByHTTPUser
      if (proxyStats.conf.subdomain) {
        this.subdomain = `${proxyStats.conf.subdomain}.${subdomainHost}`
      }
    } 
  }
}

class STCPProxy extends BaseProxy {
  constructor(proxyStats: any) {
    super(proxyStats)
    this.type = 'stcp'
  }
}

class SUDPProxy extends BaseProxy {
  constructor(proxyStats: any) {
    super(proxyStats)
    this.type = 'sudp'
  }
}

export {
  BaseProxy,
  TCPProxy,
  UDPProxy,
  TCPMuxProxy,
  HTTPProxy,
  HTTPSProxy,
  STCPProxy,
  SUDPProxy,
}
