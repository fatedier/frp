## Fixes

* Fixed control-session replacement leaks when frpc reconnects through a half-open TCP multiplexed connection.
* Fixed an SSH tunnel gateway panic when handling malformed exec requests.
