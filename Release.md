## Improvements

* Building frp from source now requires Go 1.26.0 or later.

## Fixes

* Updated dependencies to fix SSH tunnel gateway channel deadlocks and panics caused by malformed challenges from NTLM-authenticated HTTP proxies.
* Fixed VirtualNet routing issues during reconnection and shutdown, and excessive retries after repeated connection failures.
* Health checks now reset the failure count after success, so `healthCheck.maxFailed` applies to consecutive failures.
* Fixed control connections getting stuck after message write failures, which could block reconnection.
* Fixed incorrect daily traffic statistics around daylight-saving transitions and skipped local midnights.
* Fixed early-data loss in HTTP virtual-host CONNECT tunnels. Requests with a positive content length or transfer encoding now return HTTP 400.
* Fixed work connections remaining open when server-side bandwidth limiting is enabled for TCP, HTTP, and UDP proxies.
