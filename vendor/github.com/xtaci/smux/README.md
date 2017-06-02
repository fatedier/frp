<img src="smux.png" alt="smux" height="35px" />

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8] [![Coverage Statusd][9]][10]

<img src="mux.jpg" alt="smux" height="120px" /> 

[1]: https://godoc.org/github.com/xtaci/smux?status.svg
[2]: https://godoc.org/github.com/xtaci/smux
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://travis-ci.org/xtaci/smux.svg?branch=master
[6]: https://travis-ci.org/xtaci/smux
[7]: https://goreportcard.com/badge/github.com/xtaci/smux
[8]: https://goreportcard.com/report/github.com/xtaci/smux
[9]: https://codecov.io/gh/xtaci/smux/branch/master/graph/badge.svg
[10]: https://codecov.io/gh/xtaci/smux

## Introduction

Smux ( **S**imple **MU**ltiple**X**ing) is a multiplexing library for Golang. It relies on an underlying connection to provide reliability and ordering, such as TCP or [KCP](https://github.com/xtaci/kcp-go), and provides stream-oriented multiplexing. The original intention of this library is to power the connection management for [kcp-go](https://github.com/xtaci/kcp-go).

## Features

1. Tiny, less than 600 LOC.
2. ***Token bucket*** controlled receiving, which provides smoother bandwidth graph(see picture below).
3. Session-wide receive buffer, shared among streams, tightly controlled overall memory usage.
4. Minimized header(8Bytes), maximized payload. 
5. Well-tested on millions of devices in [kcptun](https://github.com/xtaci/kcptun).

![smooth bandwidth curve](curve.jpg)

## Documentation

For complete documentation, see the associated [Godoc](https://godoc.org/github.com/xtaci/smux).

## Specification

```
VERSION(1B) | CMD(1B) | LENGTH(2B) | STREAMID(4B) | DATA(LENGTH)  
```

## Usage

The API of smux are mostly taken from [yamux](https://github.com/hashicorp/yamux)

```go

func client() {
    // Get a TCP connection
    conn, err := net.Dial(...)
    if err != nil {
        panic(err)
    }

    // Setup client side of smux
    session, err := smux.Client(conn, nil)
    if err != nil {
        panic(err)
    }

    // Open a new stream
    stream, err := session.OpenStream()
    if err != nil {
        panic(err)
    }

    // Stream implements io.ReadWriteCloser
    stream.Write([]byte("ping"))
}

func server() {
    // Accept a TCP connection
    conn, err := listener.Accept()
    if err != nil {
        panic(err)
    }

    // Setup server side of smux
    session, err := smux.Server(conn, nil)
    if err != nil {
        panic(err)
    }

    // Accept a stream
    stream, err := session.AcceptStream()
    if err != nil {
        panic(err)
    }

    // Listen for a message
    buf := make([]byte, 4)
    stream.Read(buf)
}

```

## Status

Stable
