class BaseProxy {
  name: string
  type: string
  encryption: boolean
  compression: boolean
  conns: number
  traffic_in: number
  traffic_out: number
  last_start_time: string
  last_close_time: string
  status: string
  client_version: string
  addr: string
  port: number

  custom_domains: string
  host_header_rewrite: string
  locations: string
  subdomain: string

  constructor(proxyStats: any) {
    this.name = proxyStats.name
    this.type = ''
    if (proxyStats.conf != null) {
      this.encryption = proxyStats.conf.use_encryption
      this.compression = proxyStats.conf.use_compression
    } else {
      this.encryption = false
      this.compression = false
    }
    this.conns = proxyStats.cur_conns
    this.traffic_in = proxyStats.today_traffic_in
    this.traffic_out = proxyStats.today_traffic_out
    this.last_start_time = proxyStats.last_start_time
    this.last_close_time = proxyStats.last_close_time
    this.status = proxyStats.status
    this.client_version = proxyStats.client_version

    this.addr = ''
    this.port = 0
    this.custom_domains = ''
    this.host_header_rewrite = ''
    this.locations = ''
    this.subdomain = ''
  }
}

class TCPProxy extends BaseProxy {
  constructor(proxyStats: any) {
    super(proxyStats)
    this.type = 'tcp'
    if (proxyStats.conf != null) {
      this.addr = ':' + proxyStats.conf.remote_port
      this.port = proxyStats.conf.remote_port
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
      this.addr = ':' + proxyStats.conf.remote_port
      this.port = proxyStats.conf.remote_port
    } else {
      this.addr = ''
      this.port = 0
    }
  }
}

class HTTPProxy extends BaseProxy {
  constructor(proxyStats: any, port: number, subdomain_host: string) {
    super(proxyStats)
    this.type = 'http'
    this.port = port
    if (proxyStats.conf != null) {
      this.custom_domains = proxyStats.conf.custom_domains
      this.host_header_rewrite = proxyStats.conf.host_header_rewrite
      this.locations = proxyStats.conf.locations
      if (proxyStats.conf.subdomain != '') {
        this.subdomain = proxyStats.conf.subdomain + '.' + subdomain_host
      } else {
        this.subdomain = ''
      }
    } else {
      this.custom_domains = ''
      this.host_header_rewrite = ''
      this.subdomain = ''
      this.locations = ''
    }
  }
}

class HTTPSProxy extends BaseProxy {
  constructor(proxyStats: any, port: number, subdomain_host: string) {
    super(proxyStats)
    this.type = 'https'
    this.port = port
    if (proxyStats.conf != null) {
      this.custom_domains = proxyStats.conf.custom_domains
      if (proxyStats.conf.subdomain != '') {
        this.subdomain = proxyStats.conf.subdomain + '.' + subdomain_host
      } else {
        this.subdomain = ''
      }
    } else {
      this.custom_domains = ''
      this.subdomain = ''
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
