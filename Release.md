## Features

* HTTPS proxies now support load balancing groups. Multiple HTTPS proxies can be configured with the same `loadBalancer.group` and `loadBalancer.groupKey` to share the same custom domain and distribute traffic across multiple backend services, similar to the existing TCP and HTTP load balancing capabilities.
* Individual frpc proxies and visitors now accept an `enabled` flag (defaults to true), letting you disable specific entries without relying on the global `start` list—disabled blocks are skipped when client configs load.

## Improvements

* **VirtualNet**: Implemented intelligent reconnection with exponential backoff. When connection errors occur repeatedly, the reconnect interval increases from 60s to 300s (max), reducing unnecessary reconnection attempts. Normal disconnections still reconnect quickly at 10s intervals.

## Fixes

* Fix deadlock issue when TCP connection is closed. Previously, sending messages could block forever if the connection handler had already stopped.
