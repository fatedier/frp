## Features

* Add NAT traversal configuration options for XTCP proxies and visitors. Support disabling assisted addresses to avoid using slow VPN connections during NAT hole punching.
* Enhanced OIDC client configuration with support for custom TLS certificate verification and proxy settings. Added `trustedCaFile`, `insecureSkipVerify`, and `proxyURL` options for OIDC token endpoint connections.
* Added detailed Prometheus metrics with `proxy_counts_detailed` metric that includes both proxy type and proxy name labels, enabling monitoring of individual proxy connections instead of just aggregate counts.
