### Notable Changes

* The minimum supported Go version has been updated to `1.22`. In the new version of Go, the default minimum supported TLS version has been changed to `TLS 1.2`.
* The default value of `--strict-config` has been changed from `false` to `true`. If your configuration file uses a non-existent configuration item or has a spelling error, the application will throw an error. This startup parameter was introduced in version `v0.53.0`. If you wish to continue using the old behavior, you need to explicitly set `--strict-config=false`.

### Features

* Proxy supports configuring annotations, which will be displayed in the frps dashboard.

### Changes

* Removed dependencies on the forked version of kcp-go and beego log, kcp-go now uses the upstream version, and golib/log replaces beego log.
