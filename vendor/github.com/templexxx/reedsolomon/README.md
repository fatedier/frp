# Reed-Solomon

[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Build Status][5]][6] [![Go Report Card][7]][8] 

[1]: https://godoc.org/github.com/templexxx/reedsolomon?status.svg
[2]: https://godoc.org/github.com/templexxx/reedsolomon
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://travis-ci.org/templexxx/reedsolomon.svg?branch=master
[6]: https://travis-ci.org/templexxx/reedsolomon
[7]: https://goreportcard.com/badge/github.com/templexxx/reedsolomon
[8]: https://goreportcard.com/report/github.com/templexxx/reedsolomon


## Introduction:
1.  Reed-Solomon Erasure Code engine in pure Go.
2.  Super Fast: more than 10GB/s per physics core ( 10+4, 4KB per vector, Macbook Pro 2.8 GHz Intel Core i7 )

## Installation
To get the package use the standard:
```bash
go get github.com/templexxx/reedsolomon
```

## Documentation
See the associated [GoDoc](http://godoc.org/github.com/templexxx/reedsolomon)

## Specification
### GOARCH
1. All arch are supported
2. 0.1.0 need go1.9 for sync.Map in AMD64

### Math
1. Coding over in GF(2^8)
2. Primitive Polynomial: x^8 + x^4 + x^3 + x^2 + 1 (0x1d)
3. mathtool/gentbls.go : generator Primitive Polynomial and it's log table, exp table, multiply table, inverse table etc. We can get more info about how galois field work
4. mathtool/cntinverse.go : calculate how many inverse matrix will have in different RS codes config
5. Both of Cauchy and Vandermonde Matrix are supported. Vandermonde need more operations for preserving the property that any square subset of rows is invertible

### Why so fast?
These three parts will cost too much time:

1. lookup galois-field tables
2. read/write memory
3. calculate inverse matrix in the reconstruct process

SIMD will solve no.1

Cache-friendly codes will help to solve no.2 & no.3, and more, use a sync.Map for cache inverse matrix, it will help to save about 1000ns when we need same matrix. 

## Performance

Performance depends mainly on:

1. CPU instruction extension( AVX2 or SSSE3 or none )
2. number of data/parity vects
3. unit size of calculation ( see it in rs_amd64.go )
4. size of shards
5. speed of memory (waste so much time on read/write mem, :D )
6. performance of CPU
7. the way of using ( reuse memory)

And we must know the benchmark test is quite different with encoding/decoding in practice.

Because in benchmark test loops, the CPU Cache will help a lot. In practice, we must reuse the memory to make the performance become as good as the benchmark test.

Example of performance on my MacBook 2017 i7 2.8GHz. 10+4 (with 0.1.0).

### Encoding:

| Vector size | Speed (MB/S) |
|----------------|--------------|
| 1400B              |    7655.02  |
| 4KB              |       10551.37  |
| 64KB              |       9297.25 |
| 1MB              |      6829.89 |
| 16MB              |      6312.83 |

### Reconstruct (use nil to point which one need repair):

| Vector size | Speed (MB/S) |
|----------------|--------------|
| 1400B              |    4124.85  |
| 4KB              |       5715.45 |
| 64KB              |       6050.06 |
| 1MB              |      5001.21 |
| 16MB              |      5043.04 |

### ReconstructWithPos (use a position list to point which one need repair, reuse the memory):

| Vector size | Speed (MB/S) |
|----------------|--------------|
| 1400B              |    6170.24  |
| 4KB              |       9444.86 |
| 64KB              |       9311.30 |
| 1MB              |      6781.06 |
| 16MB              |      6285.34 |

**reconstruct benchmark tests here run with inverse matrix cache, if there is no cache, it will cost more time( about 1000ns)**

## Who is using this?

1. https://github.com/xtaci/kcp-go -- A Production-Grade Reliable-UDP Library for golang

## Links & Thanks
* [Klauspost ReedSolomon](https://github.com/klauspost/reedsolomon)
* [intel ISA-L](https://github.com/01org/isa-l)
* [GF SIMD] (http://www.ssrc.ucsc.edu/papers/plank-fast13.pdf)
* [asm2plan9s] (https://github.com/fwessels/asm2plan9s)
