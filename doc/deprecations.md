# Deprecations

This document tracks deprecated features and APIs that are still shipped but scheduled for removal. Maintainers should review this list before each release to decide whether any items are due for removal.

For the version compatibility policy that bounds these support windows, see the latest `Release.md`.

## Active

### Wire protocol v1

- **Deprecated since:** v0.70.0 (planned, when v2 becomes the default).
- **Removal target:** v0.78.0 or later. v0.69.0 (the last release where v1 is the default) is supported until v0.78.0 is released, so v0.77.0 is the last release that must keep v1 support.
- **Replacement:** wire protocol v2 (`transport.wireProtocol = "v2"` in frpc).
- **Code references:** v1 message types and codec under `pkg/msg/` and the protocol negotiation path in `client/` and `server/`.
- **Notes:** Removing v1 will also drop compatibility with any frpc/frps that does not negotiate v2.

### INI configuration format

- **Deprecated since:** predates this document; startup warning has been in place for several releases.
- **Removal target:** TBD.
- **Replacement:** YAML / JSON / TOML.
- **Code references:**
  - `cmd/frpc/sub/root.go` — frpc startup warning.
  - `cmd/frps/root.go` — frps startup warning.
  - `pkg/config/legacy/` — legacy INI parser; remove together with the warnings.

### Visitor connections without `runID`

- **Deprecated since:** v0.50.0 (when `runID` was introduced).
- **Removal target:** TBD.
- **Replacement:** require `runID` on every visitor connection.
- **Code references:**
  - `server/service.go` — `RegisterVisitorConn` still accepts empty `runID` for backward compatibility.
- **Notes:** Removal will break frpc clients released before v0.50.0. Schedule for a release where dropping pre-v0.50.0 frpc is acceptable.

## Removed

_None yet._
