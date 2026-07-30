## Features

* Ordinary UDP proxy payloads now use a dedicated binary codec when frpc and frps successfully negotiate the capability under wire protocol v2, reducing frame size and encoding/decoding overhead. Wire protocol v1 and SUDP remain unchanged; wire protocol v2 falls back to JSON `UDPPacket` when the peer does not support or did not negotiate the capability.
