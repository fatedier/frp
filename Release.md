## Fixes

* Fixed VirtualNet route lifecycle issues during reconnect and shutdown, including stale route cleanup, shutdown races, and reconnect backoff overflow.
* Fixed health check failure counts not resetting after a successful check, ensuring `healthCheck.maxFailed` applies to consecutive failures.
