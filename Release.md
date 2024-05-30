### Fixes

* Fixed an issue where HTTP/2 was not enabled for https2http and https2https plugins.
* Fixed the issue where the default values of INI configuration parameters are inconsistent with other configuration formats.

### Changes

* Updated the default value of `transport.tcpMuxKeepaliveInterval` from 60 to 30.
* On the Android platform, the Google DNS server is used only when the default DNS server cannot be obtained.
