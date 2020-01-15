package reedsolomon

import (
	"runtime"

	"github.com/klauspost/cpuid"
)

// Option allows to override processing parameters.
type Option func(*options)

type options struct {
	maxGoroutines                         int
	minSplitSize                          int
	useAVX512, useAVX2, useSSSE3, useSSE2 bool
	usePAR1Matrix                         bool
	useCauchy                             bool
	shardSize                             int
}

var defaultOptions = options{
	maxGoroutines: 384,
	minSplitSize:  1024,
}

func init() {
	if runtime.GOMAXPROCS(0) <= 1 {
		defaultOptions.maxGoroutines = 1
	}
	// Detect CPU capabilities.
	defaultOptions.useSSSE3 = cpuid.CPU.SSSE3()
	defaultOptions.useSSE2 = cpuid.CPU.SSE2()
	defaultOptions.useAVX2 = cpuid.CPU.AVX2()
	defaultOptions.useAVX512 = cpuid.CPU.AVX512F() && cpuid.CPU.AVX512BW()
}

// WithMaxGoroutines is the maximum number of goroutines number for encoding & decoding.
// Jobs will be split into this many parts, unless each goroutine would have to process
// less than minSplitSize bytes (set with WithMinSplitSize).
// For the best speed, keep this well above the GOMAXPROCS number for more fine grained
// scheduling.
// If n <= 0, it is ignored.
func WithMaxGoroutines(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxGoroutines = n
		}
	}
}

// WithAutoGoroutines will adjust the number of goroutines for optimal speed with a
// specific shard size.
// Send in the shard size you expect to send. Other shard sizes will work, but may not
// run at the optimal speed.
// Overwrites WithMaxGoroutines.
// If shardSize <= 0, it is ignored.
func WithAutoGoroutines(shardSize int) Option {
	return func(o *options) {
		o.shardSize = shardSize
	}
}

// WithMinSplitSize is the minimum encoding size in bytes per goroutine.
// See WithMaxGoroutines on how jobs are split.
// If n <= 0, it is ignored.
func WithMinSplitSize(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.minSplitSize = n
		}
	}
}

func withSSE3(enabled bool) Option {
	return func(o *options) {
		o.useSSSE3 = enabled
	}
}

func withAVX2(enabled bool) Option {
	return func(o *options) {
		o.useAVX2 = enabled
	}
}

func withSSE2(enabled bool) Option {
	return func(o *options) {
		o.useSSE2 = enabled
	}
}

func withAVX512(enabled bool) Option {
	return func(o *options) {
		o.useAVX512 = enabled
	}
}

// WithPAR1Matrix causes the encoder to build the matrix how PARv1
// does. Note that the method they use is buggy, and may lead to cases
// where recovery is impossible, even if there are enough parity
// shards.
func WithPAR1Matrix() Option {
	return func(o *options) {
		o.usePAR1Matrix = true
		o.useCauchy = false
	}
}

// WithCauchyMatrix will make the encoder build a Cauchy style matrix.
// The output of this is not compatible with the standard output.
// A Cauchy matrix is faster to generate. This does not affect data throughput,
// but will result in slightly faster start-up time.
func WithCauchyMatrix() Option {
	return func(o *options) {
		o.useCauchy = true
		o.usePAR1Matrix = false
	}
}
