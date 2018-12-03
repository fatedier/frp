# XOR

XOR code engine in pure Go

more than 10GB/S per core

## Introduction:

1. Use SIMD (SSE2 or AVX2) for speeding up
2. ...

## Installation
To get the package use the standard:
```bash
go get github.com/templexxx/xor
```

## Documentation

See the associated [GoDoc](http://godoc.org/github.com/templexxx/xor)


## Performance

Performance depends mainly on:

1. SIMD extension
2. unit size of worker
3. hardware ( CPU RAM etc)

Example of performance on my MacBook 2014-mid(i5-4278U 2.6GHz 2 physical cores). The 16MB per shards.
```
speed = ( shards * size ) / cost
```
| data_shards    | shard_size |speed (MB/S) |
|----------|----|-----|
| 2       |1KB|64127.95  |
|2|1400B|59657.55|
|2|16KB|35370.84|
| 2       | 16MB|12128.95 |
| 5       |1KB| 78837.33 |
|5|1400B|58054.89|
|5|16KB|50161.19|
|5| 16MB|12750.41|

## Who is using this?

1. https://github.com/xtaci/kcp-go -- A Production-Grade Reliable-UDP Library for golang