## Features

* frpc can verify server certificates using system CAs, optionally combined with CAs from `trustedCaFile`.
* `frpc reload` now refreshes certificate files for unchanged `https2http` and `https2https` proxies without interrupting existing connections. Failed refreshes keep the previous certificate and report the error in proxy status and logs while reload still succeeds.

## Fixes

* Fixed proxy and visitor configuration changes being lost when reloading frpc during reconnection.
