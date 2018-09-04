/**
 * Unit tests for ReedSolomon
 *
 * Copyright 2015, Klaus Post
 * Copyright 2015, Backblaze, Inc.  All rights reserved.
 */

package reedsolomon

import (
	"bytes"
	"math/rand"
	"runtime"
	"testing"
)

func testOpts() [][]Option {
	if !testing.Short() {
		return [][]Option{}
	}
	opts := [][]Option{
		{WithMaxGoroutines(1), WithMinSplitSize(500), withSSE3(false), withAVX2(false)},
		{WithMaxGoroutines(5000), WithMinSplitSize(50), withSSE3(false), withAVX2(false)},
		{WithMaxGoroutines(5000), WithMinSplitSize(500000), withSSE3(false), withAVX2(false)},
		{WithMaxGoroutines(1), WithMinSplitSize(500000), withSSE3(false), withAVX2(false)},
	}
	for _, o := range opts[:] {
		if defaultOptions.useSSSE3 {
			n := make([]Option, len(o), len(o)+1)
			copy(n, o)
			n = append(n, withSSE3(true))
			opts = append(opts, n)
		}
		if defaultOptions.useAVX2 {
			n := make([]Option, len(o), len(o)+1)
			copy(n, o)
			n = append(n, withAVX2(true))
			opts = append(opts, n)
		}
	}
	return opts
}

func TestEncoding(t *testing.T) {
	testEncoding(t)
	for _, o := range testOpts() {
		testEncoding(t, o...)
	}
}

func testEncoding(t *testing.T, o ...Option) {
	perShard := 50000
	r, err := New(10, 3, o...)
	if err != nil {
		t.Fatal(err)
	}
	shards := make([][]byte, 13)
	for s := range shards {
		shards[s] = make([]byte, perShard)
	}

	rand.Seed(0)
	for s := 0; s < 13; s++ {
		fillRandom(shards[s])
	}

	err = r.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := r.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	err = r.Encode(make([][]byte, 1))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	badShards := make([][]byte, 13)
	badShards[0] = make([]byte, 1)
	err = r.Encode(badShards)
	if err != ErrShardSize {
		t.Errorf("expected %v, got %v", ErrShardSize, err)
	}
}

func TestReconstruct(t *testing.T) {
	testReconstruct(t)
	for _, o := range testOpts() {
		testReconstruct(t, o...)
	}
}

func testReconstruct(t *testing.T, o ...Option) {
	perShard := 50000
	r, err := New(10, 3, o...)
	if err != nil {
		t.Fatal(err)
	}
	shards := make([][]byte, 13)
	for s := range shards {
		shards[s] = make([]byte, perShard)
	}

	rand.Seed(0)
	for s := 0; s < 13; s++ {
		fillRandom(shards[s])
	}

	err = r.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Reconstruct with all shards present
	err = r.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Reconstruct with 10 shards present
	shards[0] = nil
	shards[7] = nil
	shards[11] = nil

	err = r.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := r.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	// Reconstruct with 9 shards present (should fail)
	shards[0] = nil
	shards[4] = nil
	shards[7] = nil
	shards[11] = nil

	err = r.Reconstruct(shards)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	err = r.Reconstruct(make([][]byte, 1))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}
	err = r.Reconstruct(make([][]byte, 13))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}
}

func TestVerify(t *testing.T) {
	testVerify(t)
	for _, o := range testOpts() {
		testVerify(t, o...)
	}
}

func testVerify(t *testing.T, o ...Option) {
	perShard := 33333
	r, err := New(10, 4, o...)
	if err != nil {
		t.Fatal(err)
	}
	shards := make([][]byte, 14)
	for s := range shards {
		shards[s] = make([]byte, perShard)
	}

	rand.Seed(0)
	for s := 0; s < 10; s++ {
		fillRandom(shards[s])
	}

	err = r.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := r.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Verification failed")
	}

	// Put in random data. Verification should fail
	fillRandom(shards[10])
	ok, err = r.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Verification did not fail")
	}
	// Re-encode
	err = r.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}
	// Fill a data segment with random data
	fillRandom(shards[0])
	ok, err = r.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Verification did not fail")
	}

	_, err = r.Verify(make([][]byte, 1))
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	_, err = r.Verify(make([][]byte, 14))
	if err != ErrShardNoData {
		t.Errorf("expected %v, got %v", ErrShardNoData, err)
	}
}

func TestOneEncode(t *testing.T) {
	codec, err := New(5, 5)
	if err != nil {
		t.Fatal(err)
	}
	shards := [][]byte{
		{0, 1},
		{4, 5},
		{2, 3},
		{6, 7},
		{8, 9},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
		{0, 0},
	}
	codec.Encode(shards)
	if shards[5][0] != 12 || shards[5][1] != 13 {
		t.Fatal("shard 5 mismatch")
	}
	if shards[6][0] != 10 || shards[6][1] != 11 {
		t.Fatal("shard 6 mismatch")
	}
	if shards[7][0] != 14 || shards[7][1] != 15 {
		t.Fatal("shard 7 mismatch")
	}
	if shards[8][0] != 90 || shards[8][1] != 91 {
		t.Fatal("shard 8 mismatch")
	}
	if shards[9][0] != 94 || shards[9][1] != 95 {
		t.Fatal("shard 9 mismatch")
	}

	ok, err := codec.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("did not verify")
	}
	shards[8][0]++
	ok, err = codec.Verify(shards)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("verify did not fail as expected")
	}

}

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

func benchmarkEncode(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}
	shards := make([][]byte, dataShards+parityShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}

	b.SetBytes(int64(shardSize * dataShards))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = r.Encode(shards)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode10x2x10000(b *testing.B) {
	benchmarkEncode(b, 10, 2, 10000)
}

func BenchmarkEncode100x20x10000(b *testing.B) {
	benchmarkEncode(b, 100, 20, 10000)
}

func BenchmarkEncode17x3x1M(b *testing.B) {
	benchmarkEncode(b, 17, 3, 1024*1024)
}

// Benchmark 10 data shards and 4 parity shards with 16MB each.
func BenchmarkEncode10x4x16M(b *testing.B) {
	benchmarkEncode(b, 10, 4, 16*1024*1024)
}

// Benchmark 5 data shards and 2 parity shards with 1MB each.
func BenchmarkEncode5x2x1M(b *testing.B) {
	benchmarkEncode(b, 5, 2, 1024*1024)
}

// Benchmark 1 data shards and 2 parity shards with 1MB each.
func BenchmarkEncode10x2x1M(b *testing.B) {
	benchmarkEncode(b, 10, 2, 1024*1024)
}

// Benchmark 10 data shards and 4 parity shards with 1MB each.
func BenchmarkEncode10x4x1M(b *testing.B) {
	benchmarkEncode(b, 10, 4, 1024*1024)
}

// Benchmark 50 data shards and 20 parity shards with 1MB each.
func BenchmarkEncode50x20x1M(b *testing.B) {
	benchmarkEncode(b, 50, 20, 1024*1024)
}

// Benchmark 17 data shards and 3 parity shards with 16MB each.
func BenchmarkEncode17x3x16M(b *testing.B) {
	benchmarkEncode(b, 17, 3, 16*1024*1024)
}

func benchmarkVerify(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}
	shards := make([][]byte, parityShards+dataShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}
	err = r.Encode(shards)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(shardSize * dataShards))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = r.Verify(shards)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark 10 data slices with 2 parity slices holding 10000 bytes each
func BenchmarkVerify10x2x10000(b *testing.B) {
	benchmarkVerify(b, 10, 2, 10000)
}

// Benchmark 50 data slices with 5 parity slices holding 100000 bytes each
func BenchmarkVerify50x5x50000(b *testing.B) {
	benchmarkVerify(b, 50, 5, 100000)
}

// Benchmark 10 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkVerify10x2x1M(b *testing.B) {
	benchmarkVerify(b, 10, 2, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkVerify5x2x1M(b *testing.B) {
	benchmarkVerify(b, 5, 2, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 1MB bytes each
func BenchmarkVerify10x4x1M(b *testing.B) {
	benchmarkVerify(b, 10, 4, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkVerify50x20x1M(b *testing.B) {
	benchmarkVerify(b, 50, 20, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 16MB bytes each
func BenchmarkVerify10x4x16M(b *testing.B) {
	benchmarkVerify(b, 10, 4, 16*1024*1024)
}

func corruptRandom(shards [][]byte, dataShards, parityShards int) {
	shardsToCorrupt := rand.Intn(parityShards)
	for i := 1; i <= shardsToCorrupt; i++ {
		shards[rand.Intn(dataShards+parityShards)] = nil
	}
}

func benchmarkReconstruct(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}
	shards := make([][]byte, parityShards+dataShards)
	for s := range shards {
		shards[s] = make([]byte, shardSize)
	}

	rand.Seed(0)
	for s := 0; s < dataShards; s++ {
		fillRandom(shards[s])
	}
	err = r.Encode(shards)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(shardSize * dataShards))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		corruptRandom(shards, dataShards, parityShards)

		err = r.Reconstruct(shards)
		if err != nil {
			b.Fatal(err)
		}
		ok, err := r.Verify(shards)
		if err != nil {
			b.Fatal(err)
		}
		if !ok {
			b.Fatal("Verification failed")
		}
	}
}

// Benchmark 10 data slices with 2 parity slices holding 10000 bytes each
func BenchmarkReconstruct10x2x10000(b *testing.B) {
	benchmarkReconstruct(b, 10, 2, 10000)
}

// Benchmark 50 data slices with 5 parity slices holding 100000 bytes each
func BenchmarkReconstruct50x5x50000(b *testing.B) {
	benchmarkReconstruct(b, 50, 5, 100000)
}

// Benchmark 10 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstruct10x2x1M(b *testing.B) {
	benchmarkReconstruct(b, 10, 2, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstruct5x2x1M(b *testing.B) {
	benchmarkReconstruct(b, 5, 2, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 1MB bytes each
func BenchmarkReconstruct10x4x1M(b *testing.B) {
	benchmarkReconstruct(b, 10, 4, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstruct50x20x1M(b *testing.B) {
	benchmarkReconstruct(b, 50, 20, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 16MB bytes each
func BenchmarkReconstruct10x4x16M(b *testing.B) {
	benchmarkReconstruct(b, 10, 4, 16*1024*1024)
}

func benchmarkReconstructP(b *testing.B, dataShards, parityShards, shardSize int) {
	r, err := New(dataShards, parityShards)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(shardSize * dataShards))
	runtime.GOMAXPROCS(runtime.NumCPU())
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		shards := make([][]byte, parityShards+dataShards)
		for s := range shards {
			shards[s] = make([]byte, shardSize)
		}

		rand.Seed(0)
		for s := 0; s < dataShards; s++ {
			fillRandom(shards[s])
		}
		err = r.Encode(shards)
		if err != nil {
			b.Fatal(err)
		}

		for pb.Next() {
			corruptRandom(shards, dataShards, parityShards)

			err = r.Reconstruct(shards)
			if err != nil {
				b.Fatal(err)
			}
			ok, err := r.Verify(shards)
			if err != nil {
				b.Fatal(err)
			}
			if !ok {
				b.Fatal("Verification failed")
			}
		}
	})
}

// Benchmark 10 data slices with 2 parity slices holding 10000 bytes each
func BenchmarkReconstructP10x2x10000(b *testing.B) {
	benchmarkReconstructP(b, 10, 2, 10000)
}

// Benchmark 50 data slices with 5 parity slices holding 100000 bytes each
func BenchmarkReconstructP50x5x50000(b *testing.B) {
	benchmarkReconstructP(b, 50, 5, 100000)
}

// Benchmark 10 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstructP10x2x1M(b *testing.B) {
	benchmarkReconstructP(b, 10, 2, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstructP5x2x1M(b *testing.B) {
	benchmarkReconstructP(b, 5, 2, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 1MB bytes each
func BenchmarkReconstructP10x4x1M(b *testing.B) {
	benchmarkReconstructP(b, 10, 4, 1024*1024)
}

// Benchmark 5 data slices with 2 parity slices holding 1MB bytes each
func BenchmarkReconstructP50x20x1M(b *testing.B) {
	benchmarkReconstructP(b, 50, 20, 1024*1024)
}

// Benchmark 10 data slices with 4 parity slices holding 16MB bytes each
func BenchmarkReconstructP10x4x16M(b *testing.B) {
	benchmarkReconstructP(b, 10, 4, 16*1024*1024)
}

func TestEncoderReconstruct(t *testing.T) {
	testEncoderReconstruct(t)
	for _, o := range testOpts() {
		testEncoderReconstruct(t, o...)
	}
}

func testEncoderReconstruct(t *testing.T, o ...Option) {
	// Create some sample data
	var data = make([]byte, 250000)
	fillRandom(data)

	// Create 5 data slices of 50000 elements each
	enc, err := New(5, 3, o...)
	if err != nil {
		t.Fatal(err)
	}
	shards, err := enc.Split(data)
	if err != nil {
		t.Fatal(err)
	}
	err = enc.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err := enc.Verify(shards)
	if !ok || err != nil {
		t.Fatal("not ok:", ok, "err:", err)
	}

	// Delete a shard
	shards[0] = nil

	// Should reconstruct
	err = enc.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err = enc.Verify(shards)
	if !ok || err != nil {
		t.Fatal("not ok:", ok, "err:", err)
	}

	// Recover original bytes
	buf := new(bytes.Buffer)
	err = enc.Join(buf, shards, len(data))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("recovered bytes do not match")
	}

	// Corrupt a shard
	shards[0] = nil
	shards[1][0], shards[1][500] = 75, 75

	// Should reconstruct (but with corrupted data)
	err = enc.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err = enc.Verify(shards)
	if ok || err != nil {
		t.Fatal("error or ok:", ok, "err:", err)
	}

	// Recovered data should not match original
	buf.Reset()
	err = enc.Join(buf, shards, len(data))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(buf.Bytes(), data) {
		t.Fatal("corrupted data matches original")
	}
}

func TestSplitJoin(t *testing.T) {
	var data = make([]byte, 250000)
	rand.Seed(0)
	fillRandom(data)

	enc, _ := New(5, 3)
	shards, err := enc.Split(data)
	if err != nil {
		t.Fatal(err)
	}

	_, err = enc.Split([]byte{})
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}

	buf := new(bytes.Buffer)
	err = enc.Join(buf, shards, 50)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), data[:50]) {
		t.Fatal("recovered data does match original")
	}

	err = enc.Join(buf, [][]byte{}, 0)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	err = enc.Join(buf, shards, len(data)+1)
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}

	shards[0] = nil
	err = enc.Join(buf, shards, len(data))
	if err != ErrReconstructRequired {
		t.Errorf("expected %v, got %v", ErrReconstructRequired, err)
	}
}

func TestCodeSomeShards(t *testing.T) {
	var data = make([]byte, 250000)
	fillRandom(data)
	enc, _ := New(5, 3)
	r := enc.(*reedSolomon) // need to access private methods
	shards, _ := enc.Split(data)

	old := runtime.GOMAXPROCS(1)
	r.codeSomeShards(r.parity, shards[:r.DataShards], shards[r.DataShards:], r.ParityShards, len(shards[0]))

	// hopefully more than 1 CPU
	runtime.GOMAXPROCS(runtime.NumCPU())
	r.codeSomeShards(r.parity, shards[:r.DataShards], shards[r.DataShards:], r.ParityShards, len(shards[0]))

	// reset MAXPROCS, otherwise testing complains
	runtime.GOMAXPROCS(old)
}

func TestAllMatrices(t *testing.T) {
	t.Skip("Skipping slow matrix check")
	for i := 1; i < 257; i++ {
		_, err := New(i, i)
		if err != nil {
			t.Fatal("creating matrix size", i, i, ":", err)
		}
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		data, parity int
		err          error
	}{
		{127, 127, nil},
		{256, 256, ErrMaxShardNum},

		{0, 1, ErrInvShardNum},
		{1, 0, ErrInvShardNum},
		{257, 1, ErrMaxShardNum},

		// overflow causes r.Shards to be negative
		{256, int(^uint(0) >> 1), errInvalidRowSize},
	}
	for _, test := range tests {
		_, err := New(test.data, test.parity)
		if err != test.err {
			t.Errorf("New(%v, %v): expected %v, got %v", test.data, test.parity, test.err, err)
		}
	}
}
