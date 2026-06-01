## Features

* `transport.wireProtocol = "v2"` now also applies to UDP-based proxy payloads, including ordinary UDP and SUDP, so their payload framing is consistent with the selected wire protocol.
* Improved SUDP compatibility during mixed `transport.wireProtocol` deployments, allowing frps to bridge payloads between v1/default and v2 SUDP clients.
* XTCP work connection `NatHoleSid` messages now follow the selected `transport.wireProtocol`.

## Compatibility Notes

* When enabling `transport.wireProtocol = "v2"` for SUDP, upgrade both the proxy and visitor frpc instances first, or keep them on `v1` until both sides are upgraded.
