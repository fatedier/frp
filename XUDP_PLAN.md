# Plan: XUDP + Combo Types for the `MeoBaka/frp` fork

> Status: **DRAFT PENDING APPROVAL** — no code touched yet.
> Goal: add P2P for UDP and "combo types", specifically to enable smooth RDP (TCP 3389 + UDP 3389 both going P2P).

---

## 0. Overview & principles

Three features, in order of dependency:

| # | Feature | Nature | Difficulty |
|---|-----------|----------|--------|
| A | **`xudp`** (standalone P2P UDP) | New core feature | 🔴 High |
| B | **`xtcp+xudp`** (one shared hole carrying both TCP+UDP) | Multiplex both types over a single session | 🔴 High |
| C | **Combo sugar** (`tcp+udp`, `http+https`, `stcp+sudp`) | Syntactic sugar, split 1→N | 🟢 Low |

**Golden principle (from the source survey):** the NAT hole is already UDP, and frp already has all the code needed to carry UDP (`sudp`/`udp`). So do NOT rewrite anything about hole punching or UDP encapsulation — just **combine** the `xtcp` skeleton (hole punching + QUIC/KCP session) with the `sudp` internals (the `msg.UDPPacket` frame).

### Components REUSED AS-IS (not touched)

| Component | Location | Role |
|-----------|--------|---------|
| `nathole.Prepare/ExchangeInfo/MakeHole/PreCheck` | pkg/nathole/* | Hole punching — returns `*net.UDPConn` |
| `TunnelSession` + `QUICTunnelSession` + `KCPTunnelSession` | client/visitor/xtcp.go:321-436 | Reliable multi-stream channel over the UDP hole |
| `msg.UDPPacket` | pkg/msg/msg.go:186 | Encapsulation unit for a single datagram |
| `udp.NewUDPPacket` / `udp.GetContent` | pkg/proto/udp/udp.go:29,39 | datagram ⇄ struct |
| `udp.ForwardUserConn` | pkg/proto/udp/udp.go:43 | Loop for the local UDP socket ↔ channel |
| `udp.Forwarder` | pkg/proto/udp/udp.go:73 | Dial to the local UDP service ↔ channel |
| `msg.NewConn` / `msg.NewReadWriter` | pkg/msg/handler.go:37,79 | Length-prefixed codec over a stream |
| Server nathole signaling | server/proxy/xtcp.go, server/control.go, pkg/nathole/controller.go | **frps does not distinguish TCP/UDP** → 0 behavior changes |

---

## PHASE A — standalone `xudp`

### A.1 Config types

**File `pkg/config/v1/proxy.go`:**
- Add a constant next to line 237: `ProxyTypeXUDP ProxyType = "xudp"`
- Add to `proxyConfigTypeMap` (lines 241-250): `ProxyTypeXUDP: reflect.TypeFor[XUDPProxyConfig]()`
- Clone `XUDPProxyConfig` from `XTCPProxyConfig` (proxy.go:472-504) — **field-identical** (embeds `ProxyBaseConfig` → already provides `LocalIP`/`LocalPort`, `Secretkey`, `AllowUsers`, `NatTraversal`). Only change the type name; `MarshalToMsg`/`UnmarshalFromMsg` keep the same logic (they only carry `Sk`+`AllowUsers`).

```go
type XUDPProxyConfig struct {
	ProxyBaseConfig
	Secretkey    string   `json:"secretKey,omitempty"`
	AllowUsers   []string `json:"allowUsers,omitempty"`
	NatTraversal *NatTraversalConfig `json:"natTraversal,omitempty"`
}
// Complete/Clone/MarshalToMsg/UnmarshalFromMsg: copy exactly from XTCP
```

**File `pkg/config/v1/visitor.go`:**
- Constant next to line 77: `VisitorTypeXUDP VisitorType = "xudp"`
- `visitorConfigTypeMap` (81-85): add an entry
- Clone `XUDPVisitorConfig` from `XTCPVisitorConfig` (visitor.go:143-171). **Keep `Protocol` (quic/kcp) for the TUNNEL transport — do NOT overload it for TCP/UDP.** Keep `KeepTunnelOpen`, `MaxRetriesAnHour`, `MinRetryInterval`, `NatTraversal`.
- ⚠️ **Remove or redesign `FallbackTo`/`FallbackTimeoutMs`** — see section A.5.

> Decoding runs automatically through the registry (decode.go:44-70 proxy, :72-98 visitor) — no need to modify decode.

### A.2 Visitor `xudp` — **new file `client/visitor/xudp.go`**

Recipe: **copy `client/visitor/sudp.go` almost verbatim**, replacing the tunnel-conn source from `getNewVisitorConn()` (dial frps) with **a stream opened on the punched-hole session** (borrowed from xtcp).

Structure:
```go
type XUDPVisitor struct {
	*BaseVisitor
	cfg      *v1.XUDPVisitorConfig
	udpConn  *net.UDPConn                 // local bindAddr:bindPort  (from sudp)
	readCh   chan *msg.UDPPacket
	sendCh   chan *msg.UDPPacket
	session  TunnelSession                // from xtcp (QUIC/KCP)
	// + hole-punching fields: startTunnelCh, retry... (from xtcp)
}
```

`Run()` flow:
1. Choose the transport based on `cfg.Protocol`: `NewKCPTunnelSession()` / `NewQUICTunnelSession()` — **copy xtcp.go:54-60**.
2. `net.ResolveUDPAddr` + `net.ListenUDP(bindAddr:bindPort)` → `sv.udpConn` — **copy sudp.go:49-57** (NOT `net.Listen("tcp")` as in xtcp.go:64).
3. `go udp.ForwardUserConn(sv.udpConn, sv.readCh, sv.sendCh, packetSize)` — **copy sudp.go:65** (the local UDP socket loop).
4. `go sv.dispatcher()` — wait for the first packet → punch hole → open stream → run `worker`.

`makeNatHole()` — **copy xtcp.go:252-317 verbatim** (Prepare→ExchangeInfo→MakeHole→`session.Init(listenConn, raddr)`).

`worker(tunnelStream net.Conn)` — **copy sudp.go:113-200**:
- `payloadConn := msg.NewConn(tunnelStream, msg.NewReadWriter(tunnelStream, wireProtocol))`
- `workConnReaderFn`: `payloadConn.ReadMsg()` → `*msg.UDPPacket` → `readCh` (tunnel→user)
- `workConnSenderFn`: `sendCh` → `payloadConn.WriteMsg()` (user→tunnel)

**The ONLY difference from sudp:** replace `getNewVisitorConn()` (sudp.go:202) with `sv.session.OpenConn(ctx)` (a stream over the punched hole).

### A.3 Provider `xudp` — **new file `client/proxy/xudp.go`**

Copy `client/proxy/xtcp.go`, changing the local edge from TCP→UDP:
- `InWorkConn` → `nathole.Prepare/ExchangeInfo/MakeHole` → `listenByKCP`/`listenByQUIC`: **copy xtcp.go:57-215 verbatim**.
- **Change the per-stream handler:** xtcp.go:176 (`go pxy.HandleTCPWorkConnection(...)`) and :214 → replace with the UDP forwarder:
  ```go
  go pxy.handleUDPWorkConnection(muxConn)  // replaces HandleTCPWorkConnection
  ```
- `handleUDPWorkConnection(stream net.Conn)`: wrap `msg.NewConn`, run `udp.Forwarder(localUDPAddr, readCh, sendCh, ...)` with `localUDPAddr = net.ResolveUDPAddr(LocalIP, LocalPort)` — **pattern from client/proxy/sudp.go:79-193** (in place of `HandleTCPWorkConnection`, which uses TCP `libnet.Dial` at proxy.go:213).
- Register the factory: `RegisterProxyFactory(reflect.TypeOf(&v1.XUDPProxyConfig{}), NewXUDPProxy)` — pattern from xtcp.go:36-38.

### A.4 Server `xudp` — **new file `server/proxy/xudp.go`**

**Copy `server/proxy/xtcp.go:27-101` verbatim**, only changing `*v1.XTCPProxyConfig` → `*v1.XUDPProxyConfig`. Nothing is TCP-specific here (only `NatHoleController.ListenClient` + `writeNatHoleSid`). **Do NOT touch** server/control.go or pkg/nathole/controller.go.

### A.5 Handling `FallbackTo` (important caveat)

xtcp fallback (xtcp.go:161-183) calls `helper.TransferConn(fallbackName, userConn)` with `userConn` being **TCP**. xudp has no per-connection TCP object to transfer.
- **Phase A decision:** REMOVE `FallbackTo` from `XUDPVisitorConfig` (UDP fallback is meaningless — packet loss is normal for UDP). If hole punching fails → log a warning + retry (`MaxRetriesAnHour`). Document this clearly.

### A.6 Plumbing (mechanical, done alongside A.1)

| File | Change |
|------|-----|
| pkg/config/v1/validation/proxy.go:108-125 & 174-193 | add `case *v1.XUDPProxyConfig` |
| pkg/config/v1/validation/visitor.go:31-38 | add the xudp case |
| client/visitor/visitor.go:~99 (runtime switch) | `case *v1.XUDPVisitorConfig: return NewXUDPVisitor(...)` |
| cmd/frpc/sub/proxy.go:31-40 (`proxyTypes`), :42-46 (`visitorTypes`) | append `xudp` |
| client/http/model/proxy_definition.go & visitor_definition.go | add field + xudp case |
| web/frpc/src/types/constants.ts, proxy-converters.ts | add `'xudp'` |
| web/frps/src/utils/proxy.ts | add a display class |

### A.7 Testing Phase A
- Unit: config load `type="xudp"` + visitor `type="xudp"` decode/validate OK.
- Manual E2E: xudp provider → `localPort=3389` (or a UDP echo test), visitor `bindPort=13389`, send UDP through `127.0.0.1:13389` → see it come back correctly. Use the existing VPS infrastructure at 103.110.32.128.
- Verify successful hole-punch logs (nat hole success).

---

## PHASE B — `xtcp+xudp` merged into one shared hole (tailscale-style)

Goal: **one** proxy, **one** hole punch, carrying **both** TCP and UDP over the **same** QUIC/KCP session (multiple streams). This is where frp gets closest to L3.

### B.1 Core idea: tag each stream

QUIC/yamux are already multi-stream. We multiplex the two traffic types over a single session by **prefixing a byte tag at the head of each stream**:

```
When the VISITOR opens a new stream, it writes a first byte:
  0x01 = stream carries one TCP connection  (like current xtcp)
  0x02 = stream carries a UDP-packet channel (like xudp/sudp)
The PROVIDER reads the first byte -> route:
  0x01 -> dial TCP  localIP:localPort  + libio.Join   (HandleTCPWorkConnection)
  0x02 -> udp.Forwarder localIP:localPort(UDP)
```

Since both ends of the merged type are NEW code → we are free to define this protocol, without breaking compatibility with the existing xtcp/xudp.

### B.2 Config

New type `xtcpxudp` (or keep the UI name `xtcp+xudp` and map it internally to one constant):
```go
ProxyTypeXTCPXUDP ProxyType = "xtcpxudp"
```
- Provider: `LocalIP` + `LocalPort` are SHARED by both TCP and UDP (matches RDP: 3389 for both). Optional `LocalPortUDP` to override.
- Visitor: `BindAddr` + `BindPort` → open **both** a TCP listener **and** a UDP listener on the same port.

### B.3 Merged visitor — **new file `client/visitor/xtcpxudp.go`**

- Punch the hole once (`makeNatHole`, copy from xtcp) → one `session`.
- Open **two local front-ends in parallel**:
  - TCP listener (copy xtcp.go:63-68 + acceptLoop): each conn → `session.OpenConn` → **write tag 0x01** → `libio.Join`.
  - UDP listener (copy from A.2): one stream → `session.OpenConn` → **write tag 0x02** → pump `msg.UDPPacket`.

### B.4 Merged provider — **new file `client/proxy/xtcpxudp.go`**

- Punch hole + `listenByKCP/QUIC` (copy from xtcp) → `Accept()` stream loop.
- Each stream: **read one tag byte** → route:
  - `0x01` → `HandleTCPWorkConnection` (TCP dial, identical to xtcp)
  - `0x02` → `handleUDPWorkConnection` (UDP forwarder, from A.3)

### B.5 Merged server
- Copy `server/proxy/xtcp.go` → `server/proxy/xtcpxudp.go`, change the type. Still 0 behavior changes in frps.

### B.6 Testing Phase B
- Real RDP: provider `xtcpxudp localPort=3389`, visitor `bindPort=13389`. mstsc `127.0.0.1:13389` → reach the Desktop, and the RemoteFX UDP channel also goes through (verify via `netstat`/perfmon inside the RDP session showing UDP in use).
- Confirm only ONE hole is punched (one "nat hole success" line instead of two).

---

## PHASE C — Combo sugar (`tcp+udp`, `http+https`, `stcp+sudp`)

Pure "syntactic sugar": one config entry → multiple child proxies. Does NOT touch runtime/protocol.

### C.1 Make the combo decodable
**File `pkg/config/v1/proxy.go`:** add `ComboProxyConfig` + register it in `proxyConfigTypeMap` (otherwise decode.go:55-57 will report `unknown proxy type`).
```go
type ComboProxyConfig struct {
	ProxyBaseConfig            // name, type, localIP, localPort, sk...
	// keep enough of the union fields to generate children; or store raw and split later
}
```
Constants: `ProxyTypeTCPUDP="tcp+udp"`, `ProxyTypeHTTPHTTPS="http+https"`, `ProxyTypeSTCPSUDP="stcp+sudp"`, `ProxyTypeXTCPXUDP` (if you want this combo to also follow the sugar path — but we already did the single-hole approach in Phase B, so `xtcp+xudp` points to the merged type B and is NOT expanded).

### C.2 Expansion pass
**File `pkg/config/load.go`, insert around line ~412** (in `LoadClientConfig`, right before `CompleteProxyConfigurers`/`FilterClientConfigurers`):
```go
proxyCfgs = ExpandComboProxyConfigurers(proxyCfgs)   // 1 combo -> N child proxies
```
New function `ExpandComboProxyConfigurers([]v1.ProxyConfigurer) []v1.ProxyConfigurer`:
- Iterate; if it is a `*ComboProxyConfig` → generate children per the naming convention:
  - `name="rdp", type="tcp+udp"` → `rdp-tcp` (TCPProxyConfig) + `rdp-udp` (UDPProxyConfig), copying localIP/localPort/... to both.
  - `http+https`, `stcp+sudp`: similarly.
- Place this before Complete/Filter so children get `Complete()` + start/enabled filtering as usual.

### C.3 Child naming convention
`<name>-<subtype>` (e.g. `rdp-tcp`, `rdp-udp`). Document this to avoid name collisions.

### C.4 Testing Phase C
- `type="tcp+udp"` → after load, see exactly two proxies `-tcp` and `-udp`.
- Validate/complete runs correctly on the children.

---

## Execution order & checkpoints

```
1. Phase A.1 + A.6  (xudp config + plumbing)        -> build pass
2. Phase A.2-A.4    (xudp visitor/provider/server)  -> test UDP echo P2P
3. Phase A.7        (xudp E2E)                       -> ✅ checkpoint 1: UDP goes P2P
4. Phase B          (xtcp+xudp single hole)          -> ✅ checkpoint 2: RDP full P2P
5. Phase C          (combo sugar)                    -> ✅ checkpoint 3: concise config
```

## Risks & notes

| Risk | Handling |
|--------|-----|
| `FallbackTo` does not apply to UDP | Remove it in xudp (A.5); xtcpxudp keeps fallback for the TCP part if needed |
| `ForwardUserConn`/`Forwarder` drop packets when the channel is full | **Keep as-is** — correct UDP semantics (udp.go:66-69) |
| Hole punching fails (CGNAT/symmetric) | Retry per `MaxRetriesAnHour`; xudp has NO relay fallback → log clearly |
| Web UI / admin API hardcode the type in many places | Already listed in A.6; do it last, it does not block the core |
| Stream tag (Phase B) out of sync | Write tests for both ends; the tag is the first byte, read before routing |

## Effort estimate
- Phase A: ~4 new files + ~8 lightly modified files. Mostly copy-adapt.
- Phase B: ~3 new files (based on A).
- Phase C: ~2 modified files.
