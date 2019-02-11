class ProxyStatus {
    constructor(status) {
        this.name = status.name
        this.type = status.type
        this.status = status.status
        this.err = status.err
        this.local_addr = status.local_addr
        this.plugin = status.plugin
        this.remote_addr = status.remote_addr
    }
}

export {ProxyStatus}
