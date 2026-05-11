# Auto Transport Mode

Auto Transport Mode lets frpc and frps negotiate the transport used for the
control session when both sides explicitly opt in with `transport.protocol =
"auto"`. The client owns the final decision. The server only advertises
available endpoints and validates the client's selected endpoint.

## Scope

Implemented scope:

- client and server `transport.protocol = "auto"`
- bootstrap negotiation over TCP
- server endpoint advertisement
- client candidate filtering, probing, scoring, and selection
- control-session-level reconnect and reselection after failure
- sticky, cooldown, hysteresis, short-term blacklist, and persisted last-good
  protocol
- client `/api/transport` status API
- server `/api/serverinfo` auto transport counters
- Prometheus and in-memory server metrics
- frpc dashboard transport view and frps dashboard auto transport overview

Out of scope:

- per-proxy switching
- per-work-connection switching
- lossless migration between two control sessions
- KCP and QUIC demux on the same UDP listener
- server-pushed transport migration

## Endpoint Model

The fixed supported server-side model is:

```toml
bindPort = 7000
kcpBindPort = 7000
quicBindPort = 7002
```

This means:

- TCP uses TCP/7000
- KCP uses UDP/7000
- QUIC uses UDP/7002
- websocket and wss use the TCP bind port

TCP and KCP may share the same numeric port because they live in different
transport-layer port spaces. KCP and QUIC must not share the same UDP port in
this implementation.

## Configuration

Client:

```toml
[transport]
protocol = "auto"

[transport.auto]
enabled = true
candidates = ["quic", "tcp", "wss", "websocket", "kcp"]
allowUDP = true
strategy = "balanced"
probeTimeoutMs = 1200
probeCount = 2
stickyDurationSec = 1800
cooldownSec = 300
failureThreshold = 3
degradeThreshold = 5
recheckIntervalSec = 300
persistLastGood = true
bootstrapProtocol = "tcp"
bootstrapPort = 7000
```

Server:

```toml
[transport]
protocol = "auto"

[transport.auto]
enabled = true
allowDynamicSwitch = true
advertiseProtocols = ["tcp", "kcp", "quic", "websocket", "wss"]
preferOrder = ["quic", "tcp", "wss", "websocket", "kcp"]
switchCooldownSec = 300
```

`transport.auto.allowUDP` defaults to `true` so that `quic` and `kcp` can
participate in auto mode by default. It does not open a UDP listener on frpc.
The client only dials UDP candidates that are also advertised by frps. If
`transport.proxyURL` is configured, frpc always shrinks the candidate set to
`["tcp"]` because proxy outbound only works for TCP.

Supported client strategies:

- `balanced`: default behavior; keeps the original balanced score weights
- `latency`: gives RTT a larger penalty and server priority a smaller penalty
- `stability`: gives successful probes and last-good protocol more weight and
  penalizes prior failures more strongly

## Server Validation

When server auto mode is enabled:

- `bindPort` must be set
- `kcpBindPort` may be empty; if set, it must equal `bindPort`
- `quicBindPort` may be empty; if set, it must differ from `bindPort`
- `quicBindPort` must differ from `kcpBindPort`
- `advertiseProtocols` and `preferOrder` must contain only static transport
  protocols

If `transport.protocol` is not `auto`, the server keeps static behavior.

## Protocol Version

The auto transport protocol version is `msg.AutoTransportVersion`.

Both client and server reject messages from unsupported versions:

- `ClientHelloAuto.ClientAutoVersion`
- `SelectTransport.ClientAutoVersion`
- `ProbeTransport.ClientAutoVersion`
- `ServerHelloAuto.ServerAutoVersion`
- `ProbeTransportResp.ServerAutoVersion`

The current implementation requires an exact version match.

## Bootstrap Flow

The client always starts with a TCP bootstrap connection to `bootstrapPort`
and sends `ClientHelloAuto`.

1. frpc connects to frps over TCP bootstrap.
2. frpc sends `ClientHelloAuto`.
3. frps authenticates the message and replies with `ServerHelloAuto`.
4. frpc filters advertised endpoints against local policy.
5. frpc probes remaining candidates.
6. frpc selects the best candidate by strategy and policy.
7. frpc closes bootstrap.
8. frpc opens the selected transport for the formal control session.
9. frpc sends `SelectTransport`.
10. frpc sends the normal `Login`.
11. frps validates the selected endpoint and registers the control session.

If the server is not in auto mode or reports auto disabled, the client falls
back to static TCP and does not send `SelectTransport`.

## Messages

`ClientHelloAuto`:

```go
type ClientHelloAuto struct {
    ProtocolMode       string
    ClientCandidates   []string
    AllowUDP           bool
    HasProxyURL        bool
    TLSRequired        bool
    Strategy           string
    LastGoodProtocol   string
    BlacklistProtocols []string
    ClientAutoVersion  uint32
    PrivilegeKey       string
    Timestamp          int64
}
```

`ServerHelloAuto`:

```go
type ServerHelloAuto struct {
    ProtocolMode       string
    AutoEnabled        bool
    AllowDynamicSwitch bool
    PreferOrder        []string
    Transports         []TransportEndpoint
    ServerAutoVersion  uint32
    Error              string
}

type TransportEndpoint struct {
    Protocol string
    Addr     string
    Port     int
    Enabled  bool
}
```

`ProbeTransport` and response:

```go
type ProbeTransport struct {
    Protocol          string
    Addr              string
    Port              int
    ClientAutoVersion uint32
    PrivilegeKey      string
    Timestamp         int64
}

type ProbeTransportResp struct {
    Protocol          string
    Port              int
    ServerAutoVersion uint32
    Error             string
}
```

`SelectTransport`:

```go
type SelectTransport struct {
    Protocol          string
    Addr              string
    Port              int
    Reason            string
    Scores            map[string]int64
    ClientAutoVersion uint32
}
```

## Candidate Filtering

The client filters before probing:

- `transport.proxyURL` keeps only TCP
- `allowUDP = false` removes KCP and QUIC
- missing server endpoints are removed
- blacklisted protocols are skipped until cooldown expires
- wildcard server advertised addresses such as `0.0.0.0`, `::`, and empty
  address are resolved to the configured `serverAddr`
- when the usable endpoint address is a domain name, frpc resolves its A and
  AAAA records and probes every resolved IPv4 and IPv6 address for each
  candidate protocol. The selected control connection dials the fastest
  resolved address while preserving the original domain as the TLS server name.

If all candidates are removed only because of a local blacklist, the blacklist
is cleared once and candidates are rebuilt.

## Scoring

Each successful probe produces an `AutoTransportScoreDetail`:

```go
type AutoTransportScoreDetail struct {
    Strategy        string
    Total           int64
    Successes       int
    ProbeCount      int
    SuccessRate     float64
    AvgRTTMs        int64
    Priority        int
    SuccessScore    int64
    LatencyPenalty  int64
    PriorityPenalty int64
    LastGoodBonus   int64
    FailurePenalty  int64
}
```

The client sorts candidates by total score and applies runtime policy:

- forced reasons, such as control session close and login failure, can bypass
  sticky and cooldown
- otherwise, an existing selected protocol is kept while sticky is active
- a recently switched protocol is kept during cooldown
- a new best candidate must beat the current score by hysteresis

## Runtime Reselection

Reselection happens at control-session granularity. The client can trigger a
new selection after:

- control session close
- consecutive login failures
- consecutive heartbeat timeouts
- consecutive work connection failures
- sustained heartbeat RTT degradation

The current control session is not migrated. The client reconnects after the
failure and repeats selection.

## Observability

Client `/api/transport` returns:

- current protocol, address, port, and score
- previous and last-good protocol
- state, dynamic flag, switch count, last reason, and last error
- sticky and cooldown remaining seconds
- active blacklist
- candidate scores, score details, success rates, probe RTTs, and probe errors
- heartbeat and work connection degradation counters

Server `/api/serverinfo` returns:

- auto transport enabled flag
- advertised auto protocols
- negotiation success and failure counters
- per-protocol selection counters
- per-protocol online client counts
- switch path counters
- illegal selection counters

Prometheus exports equivalent server metrics with protocol labels.

## Acceptance Coverage

Automated coverage includes:

- real frps/frpc auto mode selects QUIC when QUIC and TCP are available
- real frps/frpc auto mode selects TCP when QUIC is not advertised as an
  endpoint
- real frps/frpc auto mode selects TCP when the client disables UDP
- real frpc falls back to static TCP when frps is not in auto mode
- `proxyURL` shrinks candidates to TCP
- `tcp@7000`, `kcp@7000`, and `quic@7002` are distinct endpoint candidates
- missing KCP and QUIC bind ports remove those endpoints
- invalid auto server port combinations fail validation
- invalid auth and unsupported auto protocol versions are rejected
- sticky, cooldown, hysteresis, blacklist, persisted last-good, and scoring
  strategies are unit tested
