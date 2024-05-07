### Notable Changes

We have optimized the heartbeat mechanism when tcpmux is enabled (enabled by default). The default value of `heartbeatInterval` has been adjusted to -1. This update ensures that when tcpmux is active, the client does not send additional heartbeats to the server. Since tcpmux incorporates its own heartbeat system, this change effectively reduces unnecessary data consumption, streamlining communication efficiency between client and server.

When connecting to frps versions older than v0.39.0 might encounter compatibility issues due to changes in the heartbeat mechanism. As a temporary workaround, setting the `heartbeatInterval` to 30 can help maintain stable connectivity with these older versions. We recommend updating to the latest frps version to leverage full functionality and improvements.

### Features

* Show tcpmux proxies on the frps dashboard.
* `http` proxy can modify the response header. For example, `responseHeaders.set.foo = "bar"` will add a new header `foo: bar` to the response.

### Fixes

* When an HTTP proxy request times out, it returns 504 instead of 404 now.
