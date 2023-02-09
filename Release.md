### New

* Added config `bandwidth_limit_mode` in frpc, default value is `client` which is current behavior. Optional value is `server`, to enable bandwidth limit in server. The major aim is to let server plugin has the ability to modify bandwidth limit for each proxy.

### Improve

* `dns_server` supports ipv6.
* frpc supports graceful shutdown for protocol `quic`.
