### Features

* Added a new plugin `tls2raw`: Enables TLS termination and forwarding of decrypted raw traffic to local service.
* Added a default timeout of 30 seconds for the frpc subcommands to prevent commands from being stuck for a long time due to network issues.

### Fixes

* Fixed the issue that when `loginFailExit = false`, the frpc stop command cannot be stopped correctly if the server is not successfully connected after startup.
