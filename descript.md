## Summary

This PR adds etcd-based multi-tenant token management for frps, enabling dynamic token configuration with per-token settings.

## Features

- **Etcd-based multi-token authentication** - Store and manage multiple tokens in etcd
- **Region validation** - Each token has a region field, must match frps's configured region
- **Per-token bandwidth limiting** - Configure bandwidth limits for individual tokens
- **Per-token port restrictions** - Limit which ports each token can use (`allowPorts`)
- **Per-token max ports** - Limit maximum number of ports per client (`maxPortsPerClient`)
- **Dynamic token management** - Add/update/delete tokens in etcd without restarting frps
- **Auto disconnect** - When a token is deleted or disabled, existing connections are automatically closed
- **Traffic reporting** - Report traffic usage to an external URL at configurable intervals per token

## Configuration

### Server (frps.toml)
```toml
bindPort = 7000

[etcd]
endpoints = ["127.0.0.1:2379"]
region = "us-east"
prefix = "/frp/tokens/"
trafficReportUrl = "http://billing-service/api/traffic/report"
```

### Token in etcd
```json
{
  "token": "your-token",
  "region": "us-east",
  "allowPorts": [{"start": 8000, "end": 9000}],
  "bandwidthLimit": "10MB",
  "maxPortsPerClient": 5,
  "enabled": true,
  "trafficReportIntervalMB": 50
}
```

### Client (frpc.toml)
No changes required - uses standard token authentication.

## Traffic Report Format

POST to configured URL:
```json
{
  "token": "xxx",
  "region": "us-east",
  "proxyName": "web",
  "trafficIn": 1048576,
  "trafficOut": 10485760,
  "timestamp": 1234567890
}
```

## Backward Compatibility

- If `[etcd]` is not configured, frps works as before with single token authentication
- No changes required for frpc

## New Files

- `pkg/config/v1/etcd.go` - EtcdConfig and TokenConfig structures
- `pkg/auth/etcd/store.go` - TokenStore with etcd client, caching, and watch
- `pkg/auth/etcd/verifier.go` - MultiTokenVerifier for authentication
- `pkg/traffic/reporter.go` - TrafficManager and TokenTrafficCounter
- `pkg/traffic/conn.go` - CountedConn wrapper for traffic counting
- `doc/etcd_multi_tenant.md` - Documentation

## Tests

- Added unit tests for `pkg/config/v1/etcd_test.go`
- Added unit tests for `pkg/traffic/reporter_test.go`
- All existing tests pass
