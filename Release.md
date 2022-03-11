### New

* Added `dial_server_timeout` in frpc to specify connect timeout to frps.
* Additional EndpointParams can be set for OIDC.
* Added CloseProxy operation in server plugin.

### Improve

* Added some randomness in reconnect delay.

### Fix

* TLS server name is ignored when `tls_trusted_ca_file` isn’t set.
