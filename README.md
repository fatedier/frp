# frp (customized fork — MeoBaka)

> This is a **customized fork** of frp (fast reverse proxy).
> For all the original documentation, architecture, and standard frp features, see the main repo:
>
> ### https://github.com/fatedier/frp
>
> This README **only describes the parts that were added/changed** compared to the original frp. Everything else
> (tcp, udp, http, https, tcpmux, stcp, sudp, xtcp, plugins, the basic dashboard, general
> installation…) works exactly like the original — read the documentation at the link above.

**Fork version:** `1.3.21.0.69.1.20 [DEV]` · **Branch:** `meobaka` (fork 1.2.0 merged onto upstream 0.69.1 line)

---

## Summary of changes

| Category | Change |
|---|---|
| New types | `xudp`, `xtcp+xudp`, `tcp+udp`, `stcp+sudp`, `mc` (Minecraft Java host routing, issue [#5390](https://github.com/fatedier/frp/issues/5390)), `pe` (Minecraft Bedrock host routing) |
| Security | Default `transport.wireProtocol` switched **v1 → v2** (v1 kept as an option) |
| Bug fix | Reconnect getting stuck after `i/o deadline reached` on v2 (issue [#5355](https://github.com/fatedier/frp/issues/5355)) |
| Dashboard | frpc admin API + Vue UI with full support for the new types |
| Examples | `examples/frpc_example.toml`, `examples/frps_example.toml` (fully commented) |

---

## 1. New types

All the types below are **real types** — fully registered on both frpc and frps. The first four merge
**TCP+UDP into a single proxy** (shown as exactly **1 proxy** on the dashboard, not "sugar" that silently
splits into multiple proxies); the last, `mc`, adds **Minecraft host-based routing**.

### `xudp` — P2P UDP NAT hole punching

The UDP counterpart of `xtcp`: the two machines punch through NAT themselves and then transfer **UDP**
directly (P2P), without going through frps. It has both a **proxy** (provider side) and a **visitor** (accessor side).

```toml
# Provider side (frpc)
[[proxies]]
name = "game-udp"
type = "xudp"
secretKey = "KEY"
localIP = "127.0.0.1"
localPort = 27015

# Accessor side (another frpc)
[[visitors]]
name = "game-udp-visitor"
type = "xudp"
serverName = "game-udp"
secretKey = "KEY"
bindAddr = "127.0.0.1"
bindPort = 27015
protocol = "quic"     # "quic" (default) or "kcp" — reliable tunnel over the UDP hole
```

> Unlike `xtcp`: `xudp` **has no** `fallbackTo` (UDP packet loss is normal).

### `xtcp+xudp` — combine TCP + UDP over **a single NAT hole** (tailscale-style)

One proxy carries **both TCP and UDP** to the local service over **the same NAT hole**. Each
stream on the tunnel is tagged with a 1-byte marker (0x01 = TCP, 0x02 = UDP) for routing. **Perfect
for Remote Desktop** (RDP uses TCP 3389 + UDP 3389 for smoothness).

**Built-in AUTO-FALLBACK:** it tries P2P hole punching first; if `fallbackTimeoutMs`
(default 1000ms) is exceeded without punching through (difficult NAT / symmetric /
CGNAT), it **automatically switches to relay via frps** (like `stcp+sudp`) — for **both TCP and
UDP**, with no extra visitor to declare. The provider pre-registers the relay path itself; the hole-punch
work-conn is marked `protocol="nathole"`, while the relay conn shares the exact same
1-byte-tag framing so the provider handles it with the same handler.

```toml
# Provider side (machine with RDP)
[[proxies]]
name = "pc-rdp"
type = "xtcp+xudp"
secretKey = "KEY_RDP"
localIP = "127.0.0.1"
localPort = 3389        # used for BOTH TCP and UDP
# localPortUDP = 3389   # (optional) change if UDP listens on a different port; default = localPort

# Accessor side
[[visitors]]
name = "pc-rdp-visitor"
type = "xtcp+xudp"
serverName = "pc-rdp"
secretKey = "KEY_RDP"
bindAddr = "127.0.0.1"
bindPort = 13389        # open mstsc → 127.0.0.1:13389 to reach the Desktop
keepTunnelOpen = true
# fallbackTimeoutMs = 1000   # (optional) how long to wait for P2P before falling back to relay
```

### `tcp+udp` — combine TCP + UDP on **a single public port**

One relay proxy opens **both TCP and UDP** on **the same `remotePort`** at frps. Mechanism:
the TCP work-conn carries an empty protocol (default), while the UDP work-conn is flagged
`"udp"` so frpc routes it to the correct handler. No visitor needed (connect straight to the public port).

```toml
[[proxies]]
name = "voice"
type = "tcp+udp"
localIP = "127.0.0.1"
localPort = 8000
localPortUDP = 8001    # (optional) different local UDP port; default = localPort
remotePort = 9000     # public TCP+UDP port on frps
```

### `stcp+sudp` — combine TCP + UDP **secretly** over relay

The secret version of `tcp+udp`: it opens no public port and is reached through a **visitor** that
holds the same `secretKey`. On frps it looks exactly like stcp/sudp (a single secret listener); the TCP/UDP
splitting lives entirely on the client via a 1-byte tag per stream (outside the encryption layer).

```toml
# Provider side
[[proxies]]
name = "priv"
type = "stcp+sudp"
secretKey = "KEY_PRIV"
localIP = "127.0.0.1"
localPort = 22
# localPortUDP = 22   # (optional) default = localPort

# Accessor side
[[visitors]]
name = "priv-visitor"
type = "stcp+sudp"
serverName = "priv"     # matches the proxy name directly
secretKey = "KEY_PRIV"
bindAddr = "127.0.0.1"
bindPort = 6100         # opens BOTH TCP and UDP at 127.0.0.1:6100
```

### `mc` — Minecraft host-based routing (Cloudflare-Spectrum style)

Route **many Minecraft (Java Edition) servers through one public port**, choosing the backend by the
**hostname in the client handshake** (the address the player typed) — exactly the way HTTPS proxies route
by TLS SNI. Inspired by [mc-gateway](https://github.com/tursom/mc-gateway); requested in issue
[#5390](https://github.com/fatedier/frp/issues/5390).

**No frps configuration at all** — the public port is declared on frpc via `remotePort`. frps opens it
lazily on first use and shares one listener among every `mc` proxy on the same port (even across
different frpc clients), releasing it when the last one disconnects. `allowPorts` still applies.

```toml
# frps.toml — NOTHING is needed for Minecraft.

# frpc.toml — each server is one proxy; reuse the same remotePort to multiplex by host.
[[proxies]]
name = "survival"
type = "mc"
localIP = "127.0.0.1"
localPort = 25565
remotePort = 25565                        # public port on frps (declared here, not on frps)
customDomains = ["survival.example.com"]  # the address players connect with

[[proxies]]
name = "creative"
type = "mc"
localPort = 25566
remotePort = 25565                        # same port -> shared listener, routed by host
customDomains = ["creative.example.com"]
```

Point each domain's DNS at the frps IP; the player just types `survival.example.com` and frps forwards
the connection — handshake intact — to the matching backend. No visitor needed; `subdomain` also works
when `subDomainHost` is set on frps.

### `pe` — Minecraft: Bedrock Edition host-routing (UDP/RakNet)

The Bedrock counterpart of `mc`. Bedrock speaks **UDP/RakNet** and only carries the hostname the player
typed **inside the login packet** (after the RakNet handshake), so — unlike Java `mc` — frps cannot peek
it. Instead **frpc runs a full Bedrock router** ([gophertunnel](https://github.com/sandertv/gophertunnel)):
it terminates each connection, reads the login `ServerAddress`, and re-originates to the matching local
server. frps only opens the public UDP port and tunnels datagrams (it treats `pe` exactly like `udp` — **no
gophertunnel on frps**). Modeled on [WaterdogPE](https://github.com/WaterdogPE/WaterdogPE) `forced_hosts`.

```toml
# frps.toml — NOTHING is needed for Bedrock.

# frpc.toml — one public UDP port, many Bedrock servers routed by hostname.
[[proxies]]
name = "bedrock"
type = "pe"
remotePort = 19132                          # public UDP port on frps
[proxies.forcedHosts]                        # hostname the player types -> local Bedrock server
"survival.example.com" = "127.0.0.1:19133"
"creative.example.com" = "127.0.0.1:19134"
```

> **Backends must run in offline mode** (`online-mode=false`): the router dials them without an Xbox
> token, WaterdogPE-style. Bedrock protocol support tracks the bundled gophertunnel version. Only `frpc`
> carries the Bedrock stack — `frps` stays clean.

**Quick comparison of the 6 types:**

| Type | Path | Visitor needed? | Used for |
|---|---|---|---|
| `xudp` | P2P NAT hole punching | Yes | Low-latency P2P UDP |
| `xtcp+xudp` | P2P NAT hole punching (1 hole, TCP+UDP), **auto-falls back to relay on difficult NAT** | Yes | Remote Desktop P2P (reliable) |
| `tcp+udp` | Relay via a public frps port | No | TCP+UDP services with a public port |
| `stcp+sudp` | Secret relay via frps | Yes | Private TCP+UDP services |
| `mc` | Host-routed relay via frps (Minecraft Java handshake) | No | Many Java servers on one public port |
| `pe` | Bedrock router on frpc (frps tunnels UDP) | No | Many Bedrock (PE) servers on one UDP port |

---

## 2. Default `wireProtocol = v2`

The `transport.wireProtocol` default is switched from **v1 → v2** because v1 has weak security. v2 uses
a new codec with AEAD encryption. The original v1 is **kept intact** and can still be re-enabled:

```toml
[transport]
wireProtocol = "v1"   # force v1 if backward compatibility is needed
```

Declare nothing → v2 runs. frps auto-detects v1/v2 via a magic number, so the two sides don't need
to be matched manually.

---

## 3. Fix for reconnect getting stuck on v2 (issue #5355)

Original symptom: *"出现 i/o deadline reached 后大概率不能重连"* — after hitting
`i/o deadline reached`, frpc logs in again with the old `runID` but keeps failing forever, forcing you to
restart frpc. This fork **resets the `runID` after 3 consecutive failed logins** so that the next
attempt is a fresh, clean login session (like a restart), enabling self-recovery.

---

## 4. Dashboard

The frpc admin API and the Vue UI (`web/frpc`) have been updated to create/edit/display the new
types (`xudp`, `xtcp+xudp`, `tcp+udp`, `stcp+sudp`) as real proxies/visitors.

---

## 5. Build

```bash
# Quick build for dev (skip embedding the web assets):
go build -tags noweb -o bin/frpc.exe ./cmd/frpc
go build -tags noweb -o bin/frps.exe ./cmd/frps

# Full build with dashboard: build the web first, then build Go:
cd web/frpc && npm install && npm run build && cd ../..
go build -o bin/frpc.exe ./cmd/frpc
go build -o bin/frps.exe ./cmd/frps
```

Binaries are emitted to the `bin/` directory. Verify the sample configs:

```bash
./bin/frpc.exe verify -c examples/frpc_example.toml
./bin/frps.exe verify -c examples/frps_example.toml
```

---

## 6. Configuration examples

`examples/frpc_example.toml` and `examples/frps_example.toml` are complete configuration sets,
with English comments on every entry — including all of the new types above.

---

*Any other features, options, and documentation not listed here: see the original repo
[fatedier/frp](https://github.com/fatedier/frp).*
