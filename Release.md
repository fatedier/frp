## Features

* When `transport.wireProtocol = "v2"` is enabled, ordinary UDP proxy work connection payloads now use wire protocol v2 message framing. This keeps UDP message payloads aligned with the negotiated frpc/frps wire protocol.

## Compatibility Notes

* The default/empty `transport.wireProtocol` and `transport.wireProtocol = "v1"` continue to use the legacy message codec for ordinary UDP proxy payloads.
* Raw stream proxy paths such as TCP, HTTP, and STCP remain unframed and are not affected by the UDP payload framing change.
* SUDP and XTCP keep their existing legacy behavior in this release and will be considered separately in a future phase.
* `transport.wireProtocol = "v2"` requires both frpc and frps to use versions that support the same wire v2 semantics. Mixing a newer peer that sends v2-framed ordinary UDP payloads with an older v2-capable peer that still expects the legacy UDP payload codec can break ordinary UDP proxy traffic.
