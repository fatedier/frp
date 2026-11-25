//go:build !frps

package proxy

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type closingWorkProxy struct {
	Proxy
	accepted atomic.Int64
}

func (p *closingWorkProxy) InWorkConn(conn net.Conn, _ *msg.StartWorkConn) {
	p.accepted.Add(1)
	_ = conn.Close()
}

func assertWrapperWorkConnClosed(t *testing.T, pw *Wrapper) {
	t.Helper()
	conn, peer := net.Pipe()
	defer conn.Close()
	defer peer.Close()
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
	pw.InWorkConn(conn, &msg.StartWorkConn{})
	_, err := peer.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

func TestWrapperInWorkConnPhaseSnapshot(t *testing.T) {
	pxy := &closingWorkProxy{}
	pw := &Wrapper{pxy: pxy, xl: xlog.New()}
	for _, phase := range []string{ProxyPhaseNew, ProxyPhaseWaitStart, ProxyPhaseStartErr, ProxyPhaseCheckFailed, ProxyPhaseClosed} {
		pw.Phase = phase
		assertWrapperWorkConnClosed(t, pw)
	}
	require.Zero(t, pxy.accepted.Load(), "inactive proxy received a work connection")
	pw.Phase = ProxyPhaseRunning
	pw.pxy = nil
	assertWrapperWorkConnClosed(t, pw)
	pw.pxy = pxy
	assertWrapperWorkConnClosed(t, pw)
	require.Equal(t, int64(1), pxy.accepted.Load())

	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		<-start
		for range 20000 {
			pw.mu.Lock()
			pw.Phase = ProxyPhaseRunning
			pw.mu.Unlock()
			pw.mu.Lock()
			pw.Phase = ProxyPhaseClosed
			pw.mu.Unlock()
		}
		close(done)
	}()
	close(start)
	for range 10000 {
		assertWrapperWorkConnClosed(t, pw)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("phase writer did not finish")
	}
}
