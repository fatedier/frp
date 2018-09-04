package reedsolomon_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"

	"github.com/klauspost/reedsolomon"
)

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

// Simple example of how to use all functions of the Encoder.
// Note that all error checks have been removed to keep it short.
func ExampleEncoder() {
	// Create some sample data
	var data = make([]byte, 250000)
	fillRandom(data)

	// Create an encoder with 17 data and 3 parity slices.
	enc, _ := reedsolomon.New(17, 3)

	// Split the data into shards
	shards, _ := enc.Split(data)

	// Encode the parity set
	_ = enc.Encode(shards)

	// Verify the parity set
	ok, _ := enc.Verify(shards)
	if ok {
		fmt.Println("ok")
	}

	// Delete two shards
	shards[10], shards[11] = nil, nil

	// Reconstruct the shards
	_ = enc.Reconstruct(shards)

	// Verify the data set
	ok, _ = enc.Verify(shards)
	if ok {
		fmt.Println("ok")
	}
	// Output: ok
	// ok
}

// This demonstrates that shards can be arbitrary sliced and
// merged and still remain valid.
func ExampleEncoder_slicing() {
	// Create some sample data
	var data = make([]byte, 250000)
	fillRandom(data)

	// Create 5 data slices of 50000 elements each
	enc, _ := reedsolomon.New(5, 3)
	shards, _ := enc.Split(data)
	err := enc.Encode(shards)
	if err != nil {
		panic(err)
	}

	// Check that it verifies
	ok, err := enc.Verify(shards)
	if ok && err == nil {
		fmt.Println("encode ok")
	}

	// Split the data set of 50000 elements into two of 25000
	splitA := make([][]byte, 8)
	splitB := make([][]byte, 8)

	// Merge into a 100000 element set
	merged := make([][]byte, 8)

	// Split/merge the shards
	for i := range shards {
		splitA[i] = shards[i][:25000]
		splitB[i] = shards[i][25000:]

		// Concencate it to itself
		merged[i] = append(make([]byte, 0, len(shards[i])*2), shards[i]...)
		merged[i] = append(merged[i], shards[i]...)
	}

	// Each part should still verify as ok.
	ok, err = enc.Verify(shards)
	if ok && err == nil {
		fmt.Println("splitA ok")
	}

	ok, err = enc.Verify(splitB)
	if ok && err == nil {
		fmt.Println("splitB ok")
	}

	ok, err = enc.Verify(merged)
	if ok && err == nil {
		fmt.Println("merge ok")
	}
	// Output: encode ok
	// splitA ok
	// splitB ok
	// merge ok
}

// This demonstrates that shards can xor'ed and
// still remain a valid set.
//
// The xor value must be the same for element 'n' in each shard,
// except if you xor with a similar sized encoded shard set.
func ExampleEncoder_xor() {
	// Create some sample data
	var data = make([]byte, 25000)
	fillRandom(data)

	// Create 5 data slices of 5000 elements each
	enc, _ := reedsolomon.New(5, 3)
	shards, _ := enc.Split(data)
	err := enc.Encode(shards)
	if err != nil {
		panic(err)
	}

	// Check that it verifies
	ok, err := enc.Verify(shards)
	if !ok || err != nil {
		fmt.Println("falied initial verify", err)
	}

	// Create an xor'ed set
	xored := make([][]byte, 8)

	// We xor by the index, so you can see that the xor can change,
	// It should however be constant vertically through your slices.
	for i := range shards {
		xored[i] = make([]byte, len(shards[i]))
		for j := range xored[i] {
			xored[i][j] = shards[i][j] ^ byte(j&0xff)
		}
	}

	// Each part should still verify as ok.
	ok, err = enc.Verify(xored)
	if ok && err == nil {
		fmt.Println("verified ok after xor")
	}
	// Output: verified ok after xor
}

// This will show a simple stream encoder where we encode from
// a []io.Reader which contain a reader for each shard.
//
// Input and output can be exchanged with files, network streams
// or what may suit your needs.
func ExampleStreamEncoder() {
	dataShards := 5
	parityShards := 2

	// Create a StreamEncoder with the number of data and
	// parity shards.
	rs, err := reedsolomon.NewStream(dataShards, parityShards)
	if err != nil {
		log.Fatal(err)
	}

	shardSize := 50000

	// Create input data shards.
	input := make([][]byte, dataShards)
	for s := range input {
		input[s] = make([]byte, shardSize)
		fillRandom(input[s])
	}

	// Convert our buffers to io.Readers
	readers := make([]io.Reader, dataShards)
	for i := range readers {
		readers[i] = io.Reader(bytes.NewBuffer(input[i]))
	}

	// Create our output io.Writers
	out := make([]io.Writer, parityShards)
	for i := range out {
		out[i] = ioutil.Discard
	}

	// Encode from input to output.
	err = rs.Encode(readers, out)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ok")
	// OUTPUT: ok
}
