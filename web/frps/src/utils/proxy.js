class BaseProxy {
    constructor(proxyStats) {
        this.name = proxyStats.name
        if (proxyStats.conf != null) {
            this.encryption = proxyStats.conf.use_encryption
            this.compression = proxyStats.conf.use_compression
        } else {
            this.encryption = ""
            this.compression = ""
        }
        this.conns = proxyStats.cur_conns
        this.traffic_in = proxyStats.today_traffic_in
        this.traffic_out = proxyStats.today_traffic_out
        this.last_start_time = proxyStats.last_start_time
        this.last_close_time = proxyStats.last_close_time
        this.status = proxyStats.status
    }
}

class TcpProxy extends BaseProxy {
    constructor(proxyStats) {
        super(proxyStats)
        this.type = "tcp"
        if (proxyStats.conf != null) {
            this.addr = ":" + proxyStats.conf.remote_port
            this.port = proxyStats.conf.remote_port
        } else {
            this.addr = ""
            this.port = ""
        }
    }
}

class UdpProxy extends BaseProxy {
    constructor(proxyStats) {
        super(proxyStats)
        this.type = "udp"
        if (proxyStats.conf != null) {
            this.addr = ":" + proxyStats.conf.remote_port
            this.port = proxyStats.conf.remote_port
        } else {
            this.addr = ""
            this.port = ""
        }
    }
}

class HttpProxy extends BaseProxy {
    constructor(proxyStats, port, subdomain_host) {
        super(proxyStats)
        this.type = "http"
        this.port = port
        if (proxyStats.conf != null) {
            this.custom_domains = proxyStats.conf.custom_domains
            this.host_header_rewrite = proxyStats.conf.host_header_rewrite
            this.locations = proxyStats.conf.locations
            if (proxyStats.conf.sub_domain != "") {
                this.subdomain = proxyStats.conf.sub_domain + "." + subdomain_host
            } else {
                this.subdomain = ""
            }
        } else {
            this.custom_domains = ""
            this.host_header_rewrite = ""
            this.subdomain = ""
            this.locations = ""
        }
    }
}

class HttpsProxy extends BaseProxy {
    constructor(proxyStats, port, subdomain_host) {
        super(proxyStats)
        this.type = "https"
        this.port = port
        if (proxyStats.conf != null) {
            this.custom_domains = proxyStats.conf.custom_domains
            if (proxyStats.conf.sub_domain != "") {
                this.subdomain = proxyStats.conf.sub_domain + "." + subdomain_host
            } else {
                this.subdomain = ""
            }
        } else {
            this.custom_domains = ""
            this.subdomain = ""
        }
    }
}

class StcpProxy extends BaseProxy {
    constructor(proxyStats) {
        super(proxyStats)
        this.type = "stcp"
    }
}

export {BaseProxy, TcpProxy, UdpProxy, HttpProxy, HttpsProxy, StcpProxy}
