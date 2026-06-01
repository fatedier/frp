## Features

* When `transport.wireProtocol = "v2"` is enabled, ordinary UDP proxy work connection payloads now use wire protocol v2 message framing. This keeps UDP message payloads aligned with the negotiated frpc/frps wire protocol.
* SUDP proxy payloads now also follow the connection wire protocol. SUDP v2 endpoints use wire protocol v2 message framing, while v1/default endpoints continue to use the legacy message codec. When the SUDP proxy frpc and visitor frpc use mixed v1/v2 wire protocols, frps bridges UDPPacket messages between the two codecs.

## Compatibility Notes

* The default/empty `transport.wireProtocol` and `transport.wireProtocol = "v1"` continue to use the legacy message codec for ordinary UDP and SUDP proxy payloads.
* Raw stream proxy paths such as TCP, HTTP, and STCP remain unframed and are not affected by the UDP/SUDP payload framing change.
* Direct NAT hole UDP sid probing packets are not changed by this release.
* `transport.wireProtocol = "v2"` requires peers to use versions that support the same wire v2 payload semantics. Mixing a newer peer that sends v2-framed UDP or SUDP payloads with an older v2-capable peer that still expects the legacy payload codec can break that proxy traffic. During rolling upgrades, upgrade both SUDP proxy and visitor frpc instances before enabling `transport.wireProtocol = "v2"` for SUDP, or keep those clients on `transport.wireProtocol = "v1"` until both sides are upgraded.
