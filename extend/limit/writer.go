package limit

import (
	"context"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Writer struct {
	w       io.Writer
	limiter *rate.Limiter
	ctx     context.Context
	mux     sync.Mutex
}

// NewWriter returns a writer that implements io.Writer with rate limiting.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   w,
		ctx: context.Background(),
		mux: sync.Mutex{},
	}
}

func NewWriterWithLimit(w io.Writer, speed uint64) *Writer {
	ww := &Writer{
		w:   w,
		ctx: context.Background(),
		mux: sync.Mutex{},
	}
	ww.SetRateLimit(speed)
	return ww
}

// SetRateLimit sets rate limit (bytes/sec) to the writer.
func (s *Writer) SetRateLimit(bytesPerSec uint64) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

// Write writes bytes from p.
func (s *Writer) Write(p []byte) (int, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.limiter == nil {
		return s.w.Write(p)
	}
	n, err := s.w.Write(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, err
}
