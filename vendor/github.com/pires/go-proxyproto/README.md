# go-proxyproto

[![Build Status](https://travis-ci.org/pires/go-proxyproto.svg?branch=master)](https://travis-ci.org/pires/go-proxyproto)
[![Coverage Status](https://coveralls.io/repos/github/pires/go-proxyproto/badge.svg?branch=master)](https://coveralls.io/github/pires/go-proxyproto?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/pires/go-proxyproto)](https://goreportcard.com/report/github.com/pires/go-proxyproto)

A Go library implementation of the [PROXY protocol, versions 1 and 2](http://www.haproxy.org/download/1.5/doc/proxy-protocol.txt),
which provides, as per specification:
> (...) a convenient way to safely transport connection
> information such as a client's address across multiple layers of NAT or TCP
> proxies. It is designed to require little changes to existing components and
> to limit the performance impact caused by the processing of the transported
> information.

This library is to be used in one of or both proxy clients and proxy servers that need to support said protocol.
Both protocol versions, 1 (text-based) and 2 (binary-based) are supported.

## Installation

```shell
$ go get -u github.com/pires/go-proxyproto
```

## Usage

### Client (TODO)

### Server (TODO)

## Documentation

[http://godoc.org/github.com/pires/go-proxyproto](http://godoc.org/github.com/pires/go-proxyproto)
