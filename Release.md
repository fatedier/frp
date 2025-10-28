## Features

* HTTPS proxies now support load balancing groups. Multiple HTTPS proxies can be configured with the same `loadBalancer.group` and `loadBalancer.groupKey` to share the same custom domain and distribute traffic across multiple backend services, similar to the existing TCP and HTTP load balancing capabilities.

## Improvements

* **VirtualNet**: Implemented intelligent reconnection with exponential backoff. When connection errors occur repeatedly, the reconnect interval increases from 60s to 300s (max), reducing unnecessary reconnection attempts. Normal disconnections still reconnect quickly at 10s intervals.
