## Features

* Support tokenSource for loading authentication tokens from files.

## Fixes

* Fix SSH tunnel gateway incorrectly binding to proxyBindAddr instead of bindAddr, which caused external connections to fail when proxyBindAddr was set to 127.0.0.1.
