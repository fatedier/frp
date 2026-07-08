## Features

* Expanded the frps dashboard API v2 migration across Clients, Proxies, Server Overview, Client Detail, and Proxy Detail, covering paginated users/clients/proxies, detail data, proxy traffic history, server system info, offline proxy statistics pruning, server-side pagination, search, and proxy type filtering.

## Fixes

* WebSocket and WSS tunnel payloads are now sent as binary frames, avoiding disconnects through RFC-compliant intermediaries that validate text frames as UTF-8.
* The `tls2raw` client plugin now writes the proxy protocol header to the local raw connection when proxy protocol is enabled.
* frpc now rejects duplicate proxy and visitor names in config files instead of silently overwriting earlier entries.
