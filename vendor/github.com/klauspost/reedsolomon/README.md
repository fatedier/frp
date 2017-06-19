# Reed-Solomon
[![GoDoc][1]][2] [![Build Status][3]][4]

[1]: https://godoc.org/github.com/klauspost/reedsolomon?status.svg
[2]: https://godoc.org/github.com/klauspost/reedsolomon
[3]: https://travis-ci.org/klauspost/reedsolomon.svg?branch=master
[4]: https://travis-ci.org/klauspost/reedsolomon

Reed-Solomon Erasure Coding in Go, with speeds exceeding 1GB/s/cpu core implemented in pure Go.

This is a golang port of the [JavaReedSolomon](https://github.com/Backblaze/JavaReedSolomon) library released by [Backblaze](http://backblaze.com), with some additional optimizations.

For an introduction on erasure coding, see the post on the [Backblaze blog](https://www.backblaze.com/blog/reed-solomon/).

Package home: https://github.com/klauspost/reedsolomon

Godoc: https://godoc.org/github.com/klauspost/reedsolomon

# Installation
To get the package use the standard:
```bash
go get github.com/klauspost/reedsolomon
```

# Usage

This section assumes you know the basics of Reed-Solomon encoding. A good start is this [Backblaze blog post](https://www.backblaze.com/blog/reed-solomon/).

This package performs the calculation of the parity sets. The usage is therefore relatively simple.

First of all, you need to choose your distribution of data and parity shards. A 'good' distribution is very subjective, and will depend a lot on your usage scenario. A good starting point is above 5 and below 257 data shards (the maximum supported number), and the number of parity shards to be 2 or above, and below the number of data shards.

To create an encoder with 10 data shards (where your data goes) and 3 parity shards (calculated):
```Go
    enc, err := reedsolomon.New(10, 3)
```
This encoder will work for all parity sets with this distribution of data and parity shards. The error will only be set if you specify 0 or negative values in any of the parameters, or if you specify more than 256 data shards.

The you send and receive data  is a simple slice of byte slices; `[][]byte`. In the example above, the top slice must have a length of 13.
```Go
    data := make([][]byte, 13)
```
You should then fill the 10 first slices with *equally sized* data, and create parity shards that will be populated with parity data. In this case we create the data in memory, but you could for instance also use [mmap](https://github.com/edsrzf/mmap-go) to map files.

```Go
    // Create all shards, size them at 50000 each
    for i := range input {
      data[i] := make([]byte, 50000)
    }
    
    
  // Fill some data into the data shards
    for i, in := range data[:10] {
      for j:= range in {
         in[j] = byte((i+j)&0xff)
      }
    }
```

To populate the parity shards, you simply call `Encode()` with your data.
```Go
    err = enc.Encode(data)
```
The only cases where you should get an error is, if the data shards aren't of equal size. The last 3 shards now contain parity data. You can verify this by calling `Verify()`:

```Go
    ok, err = enc.Verify(data)
```

The final (and important) part is to be able to reconstruct missing shards. For this to work, you need to know which parts of your data is missing. The encoder *does not know which parts are invalid*, so if data corruption is a likely scenario, you need to implement a hash check for each shard. If a byte has changed in your set, and you don't know which it is, there is no way to reconstruct the data set.

To indicate missing data, you set the shard to nil before calling `Reconstruct()`:

```Go
    // Delete two data shards
    data[3] = nil
    data[7] = nil
    
    // Reconstruct the missing shards
    err := enc.Reconstruct(data)
```
The missing data and parity shards will be recreated. If more than 3 shards are missing, the reconstruction will fail.

So to sum up reconstruction:
* The number of data/parity shards must match the numbers used for encoding.
* The order of shards must be the same as used when encoding.
* You may only supply data you know is valid.
* Invalid shards should be set to nil.

For complete examples of an encoder and decoder see the [examples folder](https://github.com/klauspost/reedsolomon/tree/master/examples).

# Splitting/Joining Data

You might have a large slice of data. To help you split this, there are some helper functions that can split and join a single byte slice.

```Go
   bigfile, _ := ioutil.Readfile("myfile.data")
   
   // Split the file
   split, err := enc.Split(bigfile)
```
This will split the file into the number of data shards set when creating the encoder and create empty parity shards. 

An important thing to note is that you have to *keep track of the exact input size*. If the size of the input isn't divisible by the number of data shards, extra zeros will be inserted in the last shard.

To join a data set, use the `Join()` function, which will join the shards and write it to the `io.Writer` you supply: 
```Go
   // Join a data set and write it to io.Discard.
   err = enc.Join(io.Discard, data, len(bigfile))
```

# Streaming/Merging

It might seem like a limitation that all data should be in memory, but an important property is that *as long as the number of data/parity shards are the same, you can merge/split data sets*, and they will remain valid as a separate set.

```Go
    // Split the data set of 50000 elements into two of 25000
    splitA := make([][]byte, 13)
    splitB := make([][]byte, 13)
    
    // Merge into a 100000 element set
    merged := make([][]byte, 13)
    
    for i := range data {
      splitA[i] = data[i][:25000]
      splitB[i] = data[i][25000:]
      
      // Concencate it to itself
	  merged[i] = append(make([]byte, 0, len(data[i])*2), data[i]...)
	  merged[i] = append(merged[i], data[i]...)
    }
    
    // Each part should still verify as ok.
    ok, err := enc.Verify(splitA)
    if ok && err == nil {
        log.Println("splitA ok")
    }
    
    ok, err = enc.Verify(splitB)
    if ok && err == nil {
        log.Println("splitB ok")
    }
    
    ok, err = enc.Verify(merge)
    if ok && err == nil {
        log.Println("merge ok")
    }
```

This means that if you have a data set that may not fit into memory, you can split processing into smaller blocks. For the best throughput, don't use too small blocks.

This also means that you can divide big input up into smaller blocks, and do reconstruction on parts of your data. This doesn't give the same flexibility of a higher number of data shards, but it will be much more performant.

# Streaming API

There has been added support for a streaming API, to help perform fully streaming operations, which enables you to do the same operations, but on streams. To use the stream API, use [`NewStream`](https://godoc.org/github.com/klauspost/reedsolomon#NewStream) function to create the encoding/decoding interfaces. You can use [`NewStreamC`](https://godoc.org/github.com/klauspost/reedsolomon#NewStreamC) to ready an interface that reads/writes concurrently from the streams.

Input is delivered as `[]io.Reader`, output as `[]io.Writer`, and functionality corresponds to the in-memory API. Each stream must supply the same amount of data, similar to how each slice must be similar size with the in-memory API. 
If an error occurs in relation to a stream, a [`StreamReadError`](https://godoc.org/github.com/klauspost/reedsolomon#StreamReadError) or [`StreamWriteError`](https://godoc.org/github.com/klauspost/reedsolomon#StreamWriteError) will help you determine which stream was the offender.

There is no buffering or timeouts/retry specified. If you want to add that, you need to add it to the Reader/Writer.

For complete examples of a streaming encoder and decoder see the [examples folder](https://github.com/klauspost/reedsolomon/tree/master/examples).

#Advanced Options

You can modify internal options which affects how jobs are split between and processed by goroutines.

To create options, use the WithXXX functions. You can supply options to `New`, `NewStream` and `NewStreamC`. If no Options are supplied, default options are used.

Example of how to supply options:

 ```Go
     enc, err := reedsolomon.New(10, 3, WithMaxGoroutines(25))
 ```


# Performance
Performance depends mainly on the number of parity shards. In rough terms, doubling the number of parity shards will double the encoding time.

Here are the throughput numbers with some different selections of data and parity shards. For reference each shard is 1MB random data, and 2 CPU cores are used for encoding.

| Data | Parity | Parity | MB/s   | SSSE3 MB/s  | SSSE3 Speed | Rel. Speed |
|------|--------|--------|--------|-------------|-------------|------------|
| 5    | 2      | 40%    | 576,11 | 2599,2      | 451%        | 100,00%    |
| 10   | 2      | 20%    | 587,73 | 3100,28     | 528%        | 102,02%    |
| 10   | 4      | 40%    | 298,38 | 2470,97     | 828%        | 51,79%     |
| 50   | 20     | 40%    | 59,81  | 713,28      | 1193%       | 10,38%     |

If `runtime.GOMAXPROCS()` is set to a value higher than 1, the encoder will use multiple goroutines to perform the calculations in `Verify`, `Encode` and `Reconstruct`.

Example of performance scaling on Intel(R) Core(TM) i7-2600 CPU @ 3.40GHz - 4 physical cores, 8 logical cores. The example uses 10 blocks with 16MB data each and 4 parity blocks.

| Threads | MB/s    | Speed |
|---------|---------|-------|
| 1       | 1355,11 | 100%  |
| 2       | 2339,78 | 172%  |
| 4       | 3179,33 | 235%  |
| 8       | 4346,18 | 321%  |

# asm2plan9s

[asm2plan9s](https://github.com/fwessels/asm2plan9s) is used for assembling the AVX2 instructions into their BYTE/WORD/LONG equivalents.

# Links
* [Backblaze Open Sources Reed-Solomon Erasure Coding Source Code](https://www.backblaze.com/blog/reed-solomon/).
* [JavaReedSolomon](https://github.com/Backblaze/JavaReedSolomon). Compatible java library by Backblaze.
* [reedsolomon-c](https://github.com/jannson/reedsolomon-c). C version, compatible with output from this package.
* [Reed-Solomon Erasure Coding in Haskell](https://github.com/NicolasT/reedsolomon). Haskell port of the package with similar performance.
* [go-erasure](https://github.com/somethingnew2-0/go-erasure). A similar library using cgo, slower in my tests.
* [rsraid](https://github.com/goayame/rsraid). A similar library written in Go. Slower, but supports more shards.
* [Screaming Fast Galois Field Arithmetic](http://www.snia.org/sites/default/files2/SDC2013/presentations/NewThinking/EthanMiller_Screaming_Fast_Galois_Field%20Arithmetic_SIMD%20Instructions.pdf). Basis for SSE3 optimizations.

# License

This code, as the original [JavaReedSolomon](https://github.com/Backblaze/JavaReedSolomon) is published under an MIT license. See LICENSE file for more information.
