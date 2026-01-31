## Features

* frpc now supports a `clientID` option to uniquely identify client instances. The server dashboard displays all connected clients with their online/offline status, connection history, and metadata, making it easier to monitor and manage multiple frpc deployments.
* Redesigned the frp web dashboard with a modern UI, dark mode support, and improved navigation.

## Fixes

* Fixed UDP proxy protocol sending header on every packet instead of only the first packet of each session.
