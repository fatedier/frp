<img src="kcp-go.png" alt="kcp-go" height="50px" />


[![GoDoc][1]][2] [![Powered][9]][10] [![MIT licensed][11]][12] [![Build Status][3]][4] [![Go Report Card][5]][6] [![Coverage Statusd][7]][8]

[1]: https://godoc.org/github.com/xtaci/kcp-go?status.svg
[2]: https://godoc.org/github.com/xtaci/kcp-go
[3]: https://travis-ci.org/xtaci/kcp-go.svg?branch=master
[4]: https://travis-ci.org/xtaci/kcp-go
[5]: https://goreportcard.com/badge/github.com/xtaci/kcp-go
[6]: https://goreportcard.com/report/github.com/xtaci/kcp-go
[7]: https://codecov.io/gh/xtaci/kcp-go/branch/master/graph/badge.svg
[8]: https://codecov.io/gh/xtaci/kcp-go
[9]: https://img.shields.io/badge/KCP-Powered-blue.svg
[10]: https://github.com/skywind3000/kcp
[11]: https://img.shields.io/badge/license-MIT-blue.svg
[12]: LICENSE

## Introduction

**kcp-go** is a **Production-Grade Reliable-UDP** library for [golang](https://golang.org/). 

It provides **fast, ordered and error-checked** delivery of streams over **UDP** packets, has been well tested with opensource project [kcptun](https://github.com/xtaci/kcptun). Millions of devices(from low-end MIPS routers to high-end servers) are running with **kcp-go** at present, including applications like **online games, live broadcasting, file synchronization and network acceleration**.

[Lastest Release](https://github.com/xtaci/kcp-go/releases)

## Features

1. Optimized for **Realtime Online Games, Audio/Video Streaming and Latency-Sensitive Distributed Consensus**.
1. Compatible with [skywind3000's](https://github.com/skywind3000) C version with language specific optimizations.
1. **Cache friendly** and **Memory optimized** design, offers extremely **High Performance** core.
1. Handles **>5K concurrent connections** on a single commodity server.
1. Compatible with [net.Conn](https://golang.org/pkg/net/#Conn) and [net.Listener](https://golang.org/pkg/net/#Listener), a drop-in replacement for [net.TCPConn](https://golang.org/pkg/net/#TCPConn).
1. [FEC(Forward Error Correction)](https://en.wikipedia.org/wiki/Forward_error_correction) Support with [Reed-Solomon Codes](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction)
1. Packet level encryption support with [AES](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard), [TEA](https://en.wikipedia.org/wiki/Tiny_Encryption_Algorithm), [3DES](https://en.wikipedia.org/wiki/Triple_DES), [Blowfish](https://en.wikipedia.org/wiki/Blowfish_(cipher)), [Cast5](https://en.wikipedia.org/wiki/CAST-128), [Salsa20]( https://en.wikipedia.org/wiki/Salsa20), etc. in [CFB](https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Cipher_Feedback_.28CFB.29) mode.
1. **Fixed number of goroutines** created for the entire server application, minimized goroutine context switch.

## Conventions

Control messages like **SYN/FIN/RST** in TCP **are not defined** in KCP, you need some **keepalive/heartbeat mechanism** in the application-level. A real world example is to use some **multiplexing** protocol over session, such as [smux](https://github.com/xtaci/smux)(with embedded keepalive mechanism), see [kcptun](https://github.com/xtaci/kcptun) for example.

## Documentation

For complete documentation, see the associated [Godoc](https://godoc.org/github.com/xtaci/kcp-go).

## Specification

<img src="frame.png" alt="Frame Format" height="109px" />

```
+-----------------+
| SESSION         |
+-----------------+
| KCP(ARQ)        |
+-----------------+
| FEC(OPTIONAL)   |
+-----------------+
| CRYPTO(OPTIONAL)|
+-----------------+
| UDP(PACKET)     |
+-----------------+
| IP              |
+-----------------+
| LINK            |
+-----------------+
| PHY             |
+-----------------+
(LAYER MODEL OF KCP-GO)
```


## Usage

Client:   [full demo](https://github.com/xtaci/kcptun/blob/master/client/main.go)
```go
kcpconn, err := kcp.DialWithOptions("192.168.0.1:10000", nil, 10, 3)
```
Server:   [full demo](https://github.com/xtaci/kcptun/blob/master/server/main.go)
```go
lis, err := kcp.ListenWithOptions(":10000", nil, 10, 3)
```

## Performance
```
  Model Name:	MacBook Pro
  Model Identifier:	MacBookPro12,1
  Processor Name:	Intel Core i5
  Processor Speed:	2.7 GHz
  Number of Processors:	1
  Total Number of Cores:	2
  L2 Cache (per Core):	256 KB
  L3 Cache:	3 MB
  Memory:	8 GB
```
```
$ go test -v -run=^$ -bench .
beginning tests, encryption:salsa20, fec:10/3
BenchmarkAES128-4          	  200000	      8256 ns/op	 363.33 MB/s	       0 B/op	       0 allocs/op
BenchmarkAES192-4          	  200000	      9153 ns/op	 327.74 MB/s	       0 B/op	       0 allocs/op
BenchmarkAES256-4          	  200000	     10079 ns/op	 297.64 MB/s	       0 B/op	       0 allocs/op
BenchmarkTEA-4             	  100000	     18643 ns/op	 160.91 MB/s	       0 B/op	       0 allocs/op
BenchmarkXOR-4             	 5000000	       316 ns/op	9486.46 MB/s	       0 B/op	       0 allocs/op
BenchmarkBlowfish-4        	   50000	     35643 ns/op	  84.17 MB/s	       0 B/op	       0 allocs/op
BenchmarkNone-4            	30000000	        56.2 ns/op	53371.83 MB/s	       0 B/op	       0 allocs/op
BenchmarkCast5-4           	   30000	     44744 ns/op	  67.05 MB/s	       0 B/op	       0 allocs/op
Benchmark3DES-4            	    2000	    639839 ns/op	   4.69 MB/s	       2 B/op	       0 allocs/op
BenchmarkTwofish-4         	   30000	     43368 ns/op	  69.17 MB/s	       0 B/op	       0 allocs/op
BenchmarkXTEA-4            	   30000	     57673 ns/op	  52.02 MB/s	       0 B/op	       0 allocs/op
BenchmarkSalsa20-4         	  300000	      3917 ns/op	 765.80 MB/s	       0 B/op	       0 allocs/op
BenchmarkFlush-4           	10000000	       226 ns/op	       0 B/op	       0 allocs/op
BenchmarkEchoSpeed4K-4     	    5000	    300030 ns/op	  13.65 MB/s	    5672 B/op	     177 allocs/op
BenchmarkEchoSpeed64K-4    	     500	   3202335 ns/op	  20.47 MB/s	   73295 B/op	    2198 allocs/op
BenchmarkEchoSpeed512K-4   	      50	  24926924 ns/op	  21.03 MB/s	  659339 B/op	   17602 allocs/op
BenchmarkEchoSpeed1M-4     	      20	  64857821 ns/op	  16.17 MB/s	 1772437 B/op	   42869 allocs/op
BenchmarkSinkSpeed4K-4     	   30000	     50230 ns/op	  81.54 MB/s	    2058 B/op	      48 allocs/op
BenchmarkSinkSpeed64K-4    	    2000	    648718 ns/op	 101.02 MB/s	   31165 B/op	     687 allocs/op
BenchmarkSinkSpeed256K-4   	     300	   4635905 ns/op	 113.09 MB/s	  286229 B/op	    5516 allocs/op
BenchmarkSinkSpeed1M-4     	     200	   9566933 ns/op	 109.60 MB/s	  463771 B/op	   10701 allocs/op
PASS
ok  	_/Users/xtaci/.godeps/src/github.com/xtaci/kcp-go	39.689s
```

## Design Considerations

1. slice vs. container/list

`kcp.flush()` loops through the send queue for retransmission checking for every 20ms(interval).

I've wrote a benchmark for comparing sequential loop through *slice* and *container/list* here:

https://github.com/xtaci/notes/blob/master/golang/benchmark2/cachemiss_test.go

```
BenchmarkLoopSlice-4   	2000000000	         0.39 ns/op
BenchmarkLoopList-4    	100000000	        54.6 ns/op
```

List structure introduces **heavy cache misses** compared to slice which owns better **locality**, 5000 connections with 32 window size and 20ms interval will cost 6us/0.03%(cpu) using slice, and 8.7ms/43.5%(cpu) for list for each `kcp.flush()`.

2. Timing accuracy vs. syscall clock_gettime

Timing is **critical** to **RTT estimator**, inaccurate timing introduces false retransmissions in KCP, but calling `time.Now()` costs 42 cycles(10.5ns on 4GHz CPU, 15.6ns on my MacBook Pro 2.7GHz), the benchmark for time.Now():

https://github.com/xtaci/notes/blob/master/golang/benchmark2/syscall_test.go

```
BenchmarkNow-4         	100000000	        15.6 ns/op
```

In kcp-go, after each `kcp.output()` function call, current time will be updated upon return, and each `kcp.flush()` will get current time once. For most of the time, 5000 connections costs 5000 * 15.6ns = 78us(no packet needs to be sent by `kcp.output()`), as for 10MB/s data transfering with 1400 MTU, `kcp.output()` will be called around 7500 times and costs 117us for `time.Now()` in **every second**.


## Tuning

Q: I'm handling >5K connections on my server. the CPU utilization is high.

A: A standalone `agent` or `gate` server for kcp-go is suggested, not only for CPU utilization, but also important to the **precision** of RTT measurements which indirectly affects retransmission. By increasing update `interval` with `SetNoDelay` like `conn.SetNoDelay(1, 40, 1, 1)` will dramatically reduce system load.

## Who is using this?

1. https://github.com/xtaci/kcptun -- A Secure Tunnel Based On KCP over UDP.
2. https://github.com/getlantern/lantern -- Lantern delivers fast access to the open Internet. 
3. https://github.com/smallnest/rpcx -- A RPC service framework based on net/rpc like alibaba Dubbo and weibo Motan.
4. https://github.com/gonet2/agent -- A gateway for games with stream multiplexing.
5. https://github.com/syncthing/syncthing -- Open Source Continuous File Synchronization.
6. https://play.google.com/store/apps/details?id=com.k17game.k3 -- Battle Zone - Earth 2048, a world-wide strategy game.

## Links

1. https://github.com/xtaci/libkcp -- FEC enhanced KCP session library for iOS/Android in C++
2. https://github.com/skywind3000/kcp -- A Fast and Reliable ARQ Protocol
3. https://github.com/templexxx/reedsolomon -- Reed-Solomon Erasure Coding in Go
