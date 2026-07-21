// Copyright 2026 The frp Authors
package limit

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestReaderZeroBurstDoesNotSpin(t *testing.T) {
	require := require.New(t)
	src := bytes.NewReader([]byte("hello"))
	// Limiter with zero burst is invalid for WaitN-based limiting; we must not hang.
	lim := rate.NewLimiter(rate.Inf, 0)
	r := NewReader(src, lim)
	buf := make([]byte, 8)
	n, err := r.Read(buf)
	require.NoError(err)
	require.Equal(5, n)
	require.Equal("hello", string(buf[:n]))
}

func TestWriterZeroBurstWritesThrough(t *testing.T) {
	require := require.New(t)
	var dst bytes.Buffer
	lim := rate.NewLimiter(rate.Inf, 0)
	w := NewWriter(&dst, lim)
	n, err := w.Write([]byte("world"))
	require.NoError(err)
	require.Equal(5, n)
	require.Equal("world", dst.String())
}

func TestReaderWriterRoundTrip(t *testing.T) {
	require := require.New(t)
	lim := rate.NewLimiter(rate.Limit(1e6), 4)
	var buf bytes.Buffer
	w := NewWriter(&buf, lim)
	_, err := io.WriteString(w, "abcdefgh")
	require.NoError(err)
	r := NewReader(bytes.NewReader(buf.Bytes()), lim)
	out, err := io.ReadAll(r)
	require.NoError(err)
	require.Equal("abcdefgh", string(out))
}
