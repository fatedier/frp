class BaseProxy {
  name: string
  type: string
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
    this.encryption = false
    this.compression = false
    if (proxyStats.conf != null && proxyStats.conf.useEncryption != null) {
      this.encryption = proxyStats.conf.useEncryption
    }
    if (proxyStats.conf != null && proxyStats.conf.useCompression != null) {
      this.compression = proxyStats.conf.useCompression
    } 
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
    if (proxyStats.conf != null) {
      if (proxyStats.conf.customDomains != null) {
        this.customDomains = proxyStats.conf.customDomains
      }
      this.hostHeaderRewrite = proxyStats.conf.hostHeaderRewrite
      this.locations = proxyStats.conf.locations
      if (proxyStats.conf.subdomain != null && proxyStats.conf.subdomain != '') {
        this.subdomain = proxyStats.conf.subdomain + '.' + subdomainHost
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
      if (proxyStats.conf.customDomains != null) {
        this.customDomains = proxyStats.conf.customDomains
      }
      if (proxyStats.conf.subdomain != null && proxyStats.conf.subdomain != '') {
        this.subdomain = proxyStats.conf.subdomain + '.' + subdomainHost
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
  HTTPProxy,
  HTTPSProxy,
  STCPProxy,
  SUDPProxy,
}
