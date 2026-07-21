## Features

* Support `maxWorkConnections` for xtcp proxies to limit concurrently handled work connections. The default 0 means no limit.

## Fixes

* Fixed control-session replacement leaks when frpc reconnects through a half-open TCP multiplexed connection.
* Fixed an SSH tunnel gateway panic when handling malformed exec requests.
