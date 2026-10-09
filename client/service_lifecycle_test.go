package client

import (
	"context"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
)

func newLifecycleTestService(t *testing.T, creator func(context.Context, *v1.ClientCommonConfig) Connector) (*Service, string) {
	t.Helper()
	svr, err := NewService(ServiceOptions{
		Common: &v1.ClientCommonConfig{
			LoginFailExit: new(false),
			WebServer:     v1.WebServerConfig{Addr: "127.0.0.1", Port: getFreeTCPPort(t)},
		},
		ConfigSourceAggregator: source.NewAggregator(source.NewConfigSource()),
		ConnectorCreator:       creator,
	})
	require.NoError(t, err)
	_, err = validation.NewConfigValidator(nil).ValidateClientCommonConfig(svr.common)
	require.NoError(t, err)
	address := svr.webServer.Address()
	assertLifecycleAdminPort(t, address, false)
	return svr, address
}

func assertLifecycleAdminPort(t *testing.T, address string, released bool) {
	t.Helper()
	ln, err := net.Listen("tcp", address)
	if ln != nil {
		_ = ln.Close()
	}
	if released {
		require.NoError(t, err, "admin listener was not released")
	} else {
		require.Error(t, err, "admin listener was not bound")
	}
}

func waitLifecycleSignal(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("service lifecycle operation did not finish")
	}
}

func TestServiceRunAfterEarlyShutdown(t *testing.T) {
	for _, mode := range []string{"close before Run", "parent already cancelled"} {
		t.Run(mode, func(t *testing.T) {
			var attempts atomic.Int64
			svr, address := newLifecycleTestService(t, func(context.Context, *v1.ClientCommonConfig) Connector {
				attempts.Add(1)
				return &failingConnector{err: net.ErrClosed}
			})
			// Detect startup's global DNS side effect without making DNS requests.
			resolver := net.DefaultResolver
			t.Cleanup(func() { net.DefaultResolver = resolver })
			svr.common.DNSServer = "127.0.0.1:53"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "close before Run" {
				svr.Close()
				assertLifecycleAdminPort(t, address, true)
			} else {
				cancel()
			}
			done := make(chan struct{})
			var runErr error
			go func() {
				runErr = svr.Run(ctx)
				close(done)
			}()
			waitLifecycleSignal(t, done)
			require.ErrorIs(t, runErr, context.Canceled)
			require.Zero(t, attempts.Load(), "closed service attempted login")
			require.Same(t, resolver, net.DefaultResolver, "closed service changed DNS")
			assertLifecycleAdminPort(t, address, true)
		})
	}
}

func TestServiceStartupShutdownLeavesCleanupToRun(t *testing.T) {
	for _, mode := range []string{"Close", "parent cancellation"} {
		t.Run(mode, func(t *testing.T) {
			entered, resume := make(chan struct{}), make(chan struct{})
			svr, address := newLifecycleTestService(t, func(ctx context.Context, _ *v1.ClientCommonConfig) Connector {
				close(entered)
				<-resume
				return &failingConnector{err: ctx.Err()}
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			defer close(resume)
			runDone := make(chan struct{})
			go func() { _ = svr.Run(ctx); close(runDone) }()
			waitLifecycleSignal(t, entered)
			closeDone := make(chan struct{})
			go func() {
				if mode == "Close" {
					svr.Close()
				} else {
					cancel()
				}
				close(closeDone)
			}()
			waitLifecycleSignal(t, closeDone)
			select {
			case <-runDone:
				t.Fatal("Run exited while startup still owned its resources")
			default:
			}
			assertLifecycleAdminPort(t, address, false)
			resume <- struct{}{}
			waitLifecycleSignal(t, runDone)
			assertLifecycleAdminPort(t, address, true)
			require.ErrorIs(t, svr.ctx.Err(), context.Canceled)
		})
	}
}

func TestServiceShutdownAfterLogin(t *testing.T) {
	for _, mode := range []string{"Close", "parent cancellation"} {
		t.Run(mode, func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			connector := &testConnector{conn: clientConn}
			svr, address := newLifecycleTestService(t, func(context.Context, *v1.ClientCommonConfig) Connector {
				return connector
			})
			defer svr.Close()
			peerDone := make(chan error, 1)
			go func() {
				rw := msg.NewV1ReadWriter(serverConn)
				var login msg.Login
				if err := rw.ReadMsgInto(&login); err != nil {
					peerDone <- err
					return
				}
				if err := rw.WriteMsg(&msg.LoginResp{RunID: "lifecycle-test"}); err != nil {
					peerDone <- err
					return
				}
				_, err := io.Copy(io.Discard, serverConn)
				peerDone <- err
			}()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			runDone := make(chan struct{})
			var runErr error
			go func() { runErr = svr.Run(ctx); close(runDone) }()
			require.Eventually(t, func() bool {
				svr.ctlMu.RLock()
				defer svr.ctlMu.RUnlock()
				return svr.ctl != nil
			}, 3*time.Second, time.Millisecond, "login did not establish control")
			if mode == "Close" {
				svr.Close()
			} else {
				cancel()
			}
			waitLifecycleSignal(t, runDone)
			require.NoError(t, runErr)
			require.True(t, connector.closed.Load())
			select {
			case err := <-peerDone:
				require.NoError(t, err)
			case <-time.After(3 * time.Second):
				t.Fatal("control connection was not released")
			}
			assertLifecycleAdminPort(t, address, true)
		})
	}
}
