### New

* Support go http pprof.

### Improve

* Change underlying TCP connection keepalive interval to 2 hours.
* Create new connection to server for `sudp` visitor when needed, to avoid frequent reconnections.
