## Compatibility Policy

Starting with v0.69.0, each minor release is supported until there are nine newer minor releases. For example, v0.69.0 will be supported until v0.78.0 is released. Within this window, frpc v0.69.0 is guaranteed to work with any frps from v0.61.0 to v0.77.0, and vice versa. Patch releases within the same minor are always compatible. Versions outside the support window may continue to work on a best-effort basis, but compatibility is no longer guaranteed.

For mixed-version deployments, upgrade frps first, then upgrade frpc. This keeps the server side ready for newer client-side protocol behavior before clients start using it.

## Notes

This release introduces wire protocol v2 as a transition path for future frpc/frps protocol changes. The existing wire protocol is difficult to extend without compatibility risk, and upcoming changes, including replacing deprecated stream encryption methods, require a versioned protocol.

**The default value of `transport.wireProtocol` remains `v1` in this release.** Users can keep the default for now. To test v2 early, upgrade both frpc and frps to versions that support it, then set `transport.wireProtocol = "v2"` in frpc. A v2-enabled frpc cannot connect to an older frps.

v1 will be deprecated when v2 becomes the default in a future release. It will continue to be supported until v0.78.0 is released, and may be removed in v0.78.0 or later.

## Features

* Added `transport.wireProtocol` for frpc to select the internal message protocol used between frpc and frps. Supported values are `v1` and `v2`.
* Added client protocol visibility in the frps dashboard and `/api/clients` API. Online clients now report their negotiated protocol as `v1` or `v2`.
