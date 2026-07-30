## Features

* Ordinary UDP proxy payloads now use a dedicated binary codec when frpc and frps successfully negotiate the capability under wire protocol v2, reducing frame size and encoding/decoding overhead. Wire protocol v1 and SUDP remain unchanged; wire protocol v2 falls back to JSON `UDPPacket` when the peer does not support or did not negotiate the capability.

## Fixes

* HTTP vhost servers no longer support HTTP/1.1 `Upgrade: h2c` requests. Cleartext HTTP/2 prior-knowledge remains supported.
* Fixed control-session replacement leaks when frpc reconnects through a half-open TCP multiplexed connection.
* Fixed an SSH tunnel gateway panic when handling malformed exec requests.
