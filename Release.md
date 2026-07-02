## Features

* Added dashboard API v2 pagination endpoints for users, clients, and proxies.
* The frps dashboard Clients and Proxies pages now use API v2 pagination and server-side search, including proxy type filtering and searchable proxy spec fields such as remote ports, custom domains, and subdomains.

## Fixes

* WebSocket and WSS tunnel payloads are now sent as binary frames, avoiding disconnects through RFC-compliant intermediaries that validate text frames as UTF-8.
* The `tls2raw` client plugin now writes the proxy protocol header to the local raw connection when proxy protocol is enabled.
* frpc now rejects duplicate proxy and visitor names in config files instead of silently overwriting earlier entries.
