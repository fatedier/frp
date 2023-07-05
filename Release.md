### Features

* frpc supports connecting to frps via the wss protocol by enabling the configuration `protocol = wss`.
* frpc supports stopping the service through the stop command.

### Improvements

* service.Run supports passing in context.

### Fixes

* Fix an issue caused by a bug in yamux that prevents wss from working properly in certain plugins.
