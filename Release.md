## Notes

**For enhanced security, the default values for `tls_enable` and `disable_custom_tls_first_byte` have been set to true.**

If you wish to revert to the previous default values, you need to manually set the values of these two parameters to false.

### Features

* Added support for `allow_users` in stcp, sudp, xtcp. By default, only the same user is allowed to access. Use `*` to allow access from any user. The visitor configuration now supports `server_user` to connect to proxies of other users.
* Added fallback support to a specified alternative visitor when xtcp connection fails.

### Improvements

* Increased the default value of `MaxStreamWindowSize` for yamux to 6MB, improving traffic forwarding rate in high-latency scenarios.

### Fixes

* Fixed an issue where having proxies with the same name would cause previously working proxies to become ineffective in `xtcp`.
