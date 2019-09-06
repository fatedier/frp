package limit

import (
	"context"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
	mux     sync.Mutex
}

// NewReader returns a reader that implements io.Reader with rate limiting.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:   r,
		ctx: context.Background(),
		mux: sync.Mutex{},
	}
}

func NewReaderWithLimit(r io.Reader, speed uint64) *Reader {
	rr := &Reader{
		r:   r,
		ctx: context.Background(),
		mux: sync.Mutex{},
	}
	rr.SetRateLimit(speed)
	return rr
}

// SetRateLimit sets rate limit (bytes/sec) to the reader.
func (s *Reader) SetRateLimit(bytesPerSec uint64) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

// Read reads bytes into p.
func (s *Reader) Read(p []byte) (int, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.limiter == nil {
		return s.r.Read(p)
	}
	n, err := s.r.Read(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, nil
}
