package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
)

type lifecycleListener struct {
	net.Listener
	accepted, closeEntered, closeResume chan struct{}
	acceptOnce, closeOnce               sync.Once
	accepts, closes                     atomic.Int64
}

func (l *lifecycleListener) Accept() (net.Conn, error) {
	l.accepts.Add(1)
	l.acceptOnce.Do(func() { close(l.accepted) })
	return l.Listener.Accept()
}

func (l *lifecycleListener) Close() error {
	l.closes.Add(1)
	err := l.Listener.Close()
	if l.closeEntered != nil {
		l.closeOnce.Do(func() { close(l.closeEntered) })
		<-l.closeResume
	}
	return err
}

func freeLifecycleTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func newLifecycleTestService(t *testing.T, web bool) (*Service, *lifecycleListener, []string) {
	t.Helper()
	cfg := &v1.ServerConfig{
		BindAddr: "127.0.0.1", ProxyBindAddr: "127.0.0.1",
	}
	if web {
		cfg.WebServer = v1.WebServerConfig{Addr: "127.0.0.1", Port: freeLifecycleTCPPort(t)}
		cfg.VhostHTTPPort = freeLifecycleTCPPort(t)
	}
	require.NoError(t, cfg.Complete())
	cfg.BindPort = 0
	_, err := validation.NewConfigValidator(nil).ValidateServerConfig(cfg)
	require.NoError(t, err)
	svr, err := NewService(cfg)
	require.NoError(t, err)
	l := &lifecycleListener{Listener: svr.listener, accepted: make(chan struct{})}
	svr.listener = l
	addresses := []string{l.Addr().String()}
	if web {
		addresses = append(addresses, svr.webServer.Address())
		addresses = append(addresses, net.JoinHostPort(cfg.ProxyBindAddr, fmt.Sprint(cfg.VhostHTTPPort)))
	}
	assertLifecyclePorts(t, addresses, false)
	return svr, l, addresses
}

func assertLifecyclePorts(t *testing.T, addresses []string, released bool) {
	t.Helper()
	for _, address := range addresses {
		ln, err := net.Listen("tcp", address)
		if ln != nil {
			_ = ln.Close()
		}
		if released {
			require.NoError(t, err, "listener %s was not released", address)
		} else {
			require.Error(t, err, "listener %s was not bound", address)
		}
	}
}

func TestServiceRunAfterEarlyShutdown(t *testing.T) {
	for _, mode := range []string{"close before Run", "close before Run with web", "parent already cancelled"} {
		t.Run(mode, func(t *testing.T) {
			svr, listener, addresses := newLifecycleTestService(t, mode == "close before Run with web")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			defer svr.Close()
			if mode != "parent already cancelled" {
				require.NoError(t, svr.Close())
				assertLifecyclePorts(t, addresses, true)
			} else {
				cancel()
			}
			done := make(chan struct{})
			go func() { svr.Run(ctx); close(done) }()
			waitForSignal(t, done, "Run after early shutdown")
			require.Zero(t, listener.accepts.Load(), "closed service started accepting")
			require.Equal(t, int64(1), listener.closes.Load())
			assertLifecyclePorts(t, addresses, true)
		})
	}
}

func TestServiceCloseMarksClosedBeforeCleanup(t *testing.T) {
	svr, listener, addresses := newLifecycleTestService(t, false)
	listener.closeEntered, listener.closeResume = make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(listener.closeResume) }) }
	defer release()
	closeDone := make(chan struct{})
	go func() { _ = svr.Close(); close(closeDone) }()
	waitForSignal(t, listener.closeEntered, "Close cleanup barrier")
	runDone := make(chan struct{})
	go func() { svr.Run(context.Background()); close(runDone) }()
	waitForSignal(t, runDone, "Run to observe closed before cleanup finishes")
	require.Zero(t, listener.accepts.Load(), "Run started during cleanup")
	assertLifecyclePorts(t, addresses, false)
	release()
	waitForSignal(t, closeDone, "Close cleanup to finish")
	assertLifecyclePorts(t, addresses, true)
}

func TestServiceParentCancellationAndCloseShareCleanup(t *testing.T) {
	svr, listener, addresses := newLifecycleTestService(t, false)
	listener.closeEntered, listener.closeResume = make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(listener.closeResume) }) }
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan struct{})
	go func() { svr.Run(ctx); close(runDone) }()
	waitForSignal(t, listener.accepted, "main listener Accept")
	cancel()
	waitForSignal(t, listener.closeEntered, "parent cancellation to close main listener")
	closeDone := make(chan struct{})
	go func() { _ = svr.Close(); close(closeDone) }()
	release()
	waitForSignal(t, closeDone, "external Close to finish")
	waitForSignal(t, runDone, "Run cleanup to finish")
	require.Equal(t, int64(1), listener.closes.Load(), "cleanup ran more than once")
	assertLifecyclePorts(t, addresses, true)
}
