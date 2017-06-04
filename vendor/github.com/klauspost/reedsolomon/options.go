package reedsolomon

import (
	"runtime"

	"github.com/klauspost/cpuid"
)

// Option allows to override processing parameters.
type Option func(*options)

type options struct {
	maxGoroutines     int
	minSplitSize      int
	useAVX2, useSSSE3 bool
}

var defaultOptions = options{
	maxGoroutines: 50,
	minSplitSize:  512,
}

func init() {
	if runtime.GOMAXPROCS(0) <= 1 {
		defaultOptions.maxGoroutines = 1
	}
	// Detect CPU capabilities.
	defaultOptions.useSSSE3 = cpuid.CPU.SSSE3()
	defaultOptions.useAVX2 = cpuid.CPU.AVX2()
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

// MinSplitSize Is the minimum encoding size in bytes per goroutine.
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
