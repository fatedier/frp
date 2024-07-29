### Features

* Added a new plugin `tls2raw`: Enables TLS termination and forwarding of decrypted raw traffic to local service.

* Fixed the issue that when `loginFailExit = false`, the frpc stop command cannot be stopped correctly if the server is not successfully connected after startup.
