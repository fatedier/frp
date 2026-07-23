## Fixes

* HTTP vhost servers no longer support HTTP/1.1 `Upgrade: h2c` requests. Cleartext HTTP/2 prior-knowledge remains supported.
* Fixed control-session replacement leaks when frpc reconnects through a half-open TCP multiplexed connection.
* Fixed an SSH tunnel gateway panic when handling malformed exec requests.
