So far, there is no mature Go project that does well in parsing `*.ini` files. 

By comparison, we have selected an open source project: `https://github.com/go-ini/ini`.

This library helped us solve most of the key-value matching, but there are still some problems, such as not supporting parsing `map`.

We add our own logic on the basis of this library. In the current situationwhich, we need to complete the entire `Unmarshal` in two steps:

* Step#1, use `go-ini` to complete the basic parameter matching;
* Step#2, parse our custom parameters to realize parsing special structure, like `map`, `array`.

Some of the keywords in `tag`(like inline, extends, etc.) may be different from standard libraries such as `json` and `protobuf` in Go. For details, please refer to the library documentation: https://ini.unknwon.io/docs/intro.