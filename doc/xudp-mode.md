# XUDP Mode: NAT Traversal with Multi-STUN Prediction

## Overview

`xudp` is a new proxy type that extends frp's NAT hole-punching capabilities to handle **predictable Symmetric NATs** more reliably. It builds on the existing `xtcp` infrastructure with three key improvements:

1. **Multi-STUN Prediction** — Queries multiple STUN servers serially from the same local port to detect port allocation patterns, then predicts the next ports the NAT will assign.
2. **Realm Rendezvous** — The frps server acts as a lightweight signaling relay using a shared token to exchange peer addresses, then cleanly exits the data flow.
3. **Decoupled Tunnel Lifecycle** — The resulting QUIC tunnel runs in an isolated goroutine with an independent context, surviving primary frpc-frps control channel drops.

## Problem Statement

The existing `xtcp` mode works well for Easy NATs (where the mapped port stays constant across destinations) but struggles with Symmetric NATs that increment ports predictably (e.g., +2 per new destination). While `xtcp` mode 1-4 behaviors attempt to handle Hard NATs, they rely on brute-force port scanning which is slow and unreliable.

Many enterprise and carrier-grade NATs exhibit **predictable port allocation** — the external port increments by a fixed delta for each new UDP flow. If we can observe this delta, we can predict where the next hole-punch packet will land.

## Architecture

```
┌─────────┐         ┌─────────┐         ┌─────────┐
│  Visitor │         │  frps   │         │  Client  │
│  (xudp)  │         │(rendezvous)│      │  (xudp)  │
└────┬─────┘         └────┬─────┘         └────┬─────┘
     │                     │                     │
     │ 1. Multi-STUN       │                     │ 1. Multi-STUN
     │    Discovery        │                     │    Discovery
     │                     │                     │
     │ 2. NatHoleVisitor ──┤                     │
     │    (addresses+proto)│                     │
     │                     │── 3. SID via ──────►│
     │                     │   work conn         │
     │                     │                     │
     │                     │◄── 4. NatHoleClient─│
     │                     │    (addresses)      │
     │                     │                     │
     │◄─ 5. NatHoleResp ──┤── 5. NatHoleResp ──►│
     │   (candidates+      │   (candidates+      │
     │    behavior)        │    behavior)        │
     │                     │                     │
     │   ═══ frps exits data flow ═══           │
     │                     │                     │
     │ 6. XUDPMakeHole ◄──────────────────────►│ 6. XUDPMakeHole
     │    (with predicted ports)                 │    (with predicted ports)
     │                     │                     │
     │ 7. QUIC Dial ◄─────────────────────────►│ 7. QUIC Listen
     │    (frp-xudp ALPN)  │                     │    (frp-xudp ALPN)
     │                     │                     │
     │◄═══════ Decoupled QUIC Tunnel ══════════►│
     │   (survives control channel drops)        │
```

## Multi-STUN Prediction

### How It Works

Standard STUN discovery queries one server and gets one external address. Multi-STUN discovery queries **N servers serially** from the **same local port**:

```
Local Port 12345 ──► STUN Server A ──► Observes external port 40000
                 ──► STUN Server B ──► Observes external port 40002
                 ──► STUN Server C ──► Observes external port 40004
```

From this sequence, we compute `portDelta = 2`. The next allocation will likely be port `40006`.

### Port Prediction Algorithm

```go
func computePortDelta(addrs []string) int {
    ports := parsePorts(addrs)
    if len(ports) < 2 {
        return 0
    }
    delta := ports[1] - ports[0]
    if delta <= 0 {
        return 0
    }
    // Verify consistency across all observations
    for i := 2; i < len(ports); i++ {
        if ports[i]-ports[i-1] != delta {
            return 0  // Not predictable
        }
    }
    if delta >= 1 && delta <= 100 {
        return delta
    }
    return 0
}
```

When a positive delta is detected, `XUDPMakeHole` appends predicted port addresses to the candidate list before invoking the standard `MakeHole` mechanism:

```go
predicted := predictNextPorts(resp.CandidateAddrs, portDelta, 3)
resp.CandidateAddrs = append(resp.CandidateAddrs, predicted...)
```

This means the hole-punch packets target both the observed ports AND the predicted next ports, significantly increasing success probability.

### Graceful Degradation

If multi-STUN prediction fails (e.g., only one STUN server configured, or no consistent delta detected), xudp falls back to the standard `MakeHole` behavior — identical to xtcp. The prediction is purely additive.

## Decoupled Tunnel Lifecycle

### The Problem

In standard `xtcp`, the QUIC/KCP tunnel is tied to the control channel's context. If the frpc-frps control connection drops (network blip, server restart), the tunnel dies even though the direct peer-to-peer UDP path is still valid.

### The Solution

The xudp tunnel uses `context.Background()` instead of the control channel's context:

```go
// Client proxy side - decoupled goroutine
func (pxy *XUDPProxy) listenByQUICDecoupled(...) {
    tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
    defer tunnelCancel()
    // ... QUIC listener runs independently
}

// Visitor side - decoupled session
quicConn, err := quic.Dial(context.Background(), listenConn, raddr, tlsConfig, ...)
```

The tunnel identifies itself with ALPN protocol `frp-xudp` to distinguish from standard xtcp QUIC connections.

### Lifecycle Guarantees

| Event | xtcp Behavior | xudp Behavior |
|-------|--------------|---------------|
| Control channel drops | Tunnel dies | Tunnel survives |
| Peer goes offline | Tunnel times out (QUIC idle) | Same |
| Explicit close | Tunnel closes | Tunnel closes |
| frpc restart | New hole punch needed | New hole punch needed |

## Configuration

### Client (frpc) Proxy

```toml
[[proxies]]
name = "p2p-xudp"
type = "xudp"
secretKey = "my-secret"
allowUsers = ["*"]

# Optional: additional STUN servers for multi-STUN prediction
stunServers = ["stun.l.google.com:19302", "stun1.l.google.com:19302"]

# Optional: disable assisted addresses
[proxies.natTraversal]
disableAssistedAddrs = false
```

### Visitor (frpc)

```toml
[[visitors]]
name = "p2p-xudp-visitor"
type = "xudp"
serverName = "p2p-xudp"
secretKey = "my-secret"
bindAddr = "127.0.0.1"
bindPort = 6000

# Optional: additional STUN servers
stunServers = ["stun.l.google.com:19302", "stun1.l.google.com:19302"]

# Tunnel management
keepTunnelOpen = true
maxRetriesAnHour = 8
minRetryInterval = 90

# Fallback to stcp if hole punch fails
fallbackTo = "stcp-visitor"
fallbackTimeoutMs = 1500
```

## When to Use xudp vs xtcp

| Scenario | Recommendation |
|----------|---------------|
| Both peers behind Easy NAT | `xtcp` (simpler, proven) |
| One peer behind Symmetric NAT with predictable ports | `xudp` |
| Both peers behind Symmetric NAT | `xudp` (best effort) |
| Need tunnel to survive control drops | `xudp` |
| Minimal configuration preferred | `xtcp` |

## Implementation Details

### Package Layout

- `pkg/nathole/xudp.go` — `DiscoverMultiSTUN`, `XUDPRendezvousExchange`, `XUDPMakeHole`, port prediction
- `pkg/nathole/xudp_tunnel.go` — `XUDPTunnel` struct (decoupled QUIC tunnel abstraction)
- `pkg/config/v1/proxy.go` — `XUDPProxyConfig` type
- `pkg/config/v1/visitor.go` — `XUDPVisitorConfig` type
- `client/proxy/xudp.go` — Client-side proxy (hole punch + QUIC listener)
- `client/visitor/xudp.go` — Visitor (tunnel session + connection handling)
- `server/proxy/xudp.go` — Server-side proxy (rendezvous relay)

### Concurrency Model

xudp reuses the existing nathole concurrency patterns:
- Unbuffered `chan string` for SID delivery (server → client)
- Buffered `chan struct{}` (cap 1) for session notification
- `select` with `time.After` for all timeout paths
- `sync.RWMutex` for tunnel session state
- `errgroup.Group` for parallel response delivery

### Wire Protocol

xudp reuses all existing message types (`NatHoleVisitor`, `NatHoleClient`, `NatHoleResp`, `NatHoleSid`, `NatHoleReport`). No new wire protocol messages are needed. The only distinction is the QUIC ALPN: `frp-xudp` vs `frp` for xtcp.
