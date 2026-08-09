## Features

* UDP packet payloads for ordinary UDP proxies and SUDP now use a dedicated binary codec when frpc and frps successfully negotiate the capability under wire protocol v2, using a more compact wire representation. Wire protocol v1 remains JSON; wire protocol v2 falls back to JSON `UDPPacket` when the peer does not support or did not negotiate the capability.

## Fixes

* Fixed a server panic and remote denial of service caused by a client sending a negative `pool_count`. Negative values are now rejected before work-connection pool resources are allocated.
* Fixed `frpc verify` ignoring configured `featureGates`, which caused VirtualNet configurations to be rejected even when the feature was enabled.
* Fixed a case-insensitive validation bypass that allowed `customDomains` under the configured `subDomainHost` to be registered using mixed-case domain names.
