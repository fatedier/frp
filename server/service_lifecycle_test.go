package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
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

func TestNewServiceDoesNotLeakWebServerOnBindFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	cfg := &v1.ServerConfig{
		BindAddr:      "127.0.0.1",
		BindPort:      ln.Addr().(*net.TCPAddr).Port,
		ProxyBindAddr: "127.0.0.1",
		WebServer:     v1.WebServerConfig{Addr: "127.0.0.1", Port: freeLifecycleTCPPort(t)},
	}
	require.NoError(t, cfg.Complete())

	svr, err := NewService(cfg)
	if svr != nil {
		defer svr.Close()
	}
	require.ErrorContains(t, err, "create server listener error")
	require.ErrorContains(t, err, ln.Addr().String())
	require.Nil(t, svr)
	assertLifecyclePorts(t, []string{net.JoinHostPort(cfg.WebServer.Addr, fmt.Sprint(cfg.WebServer.Port))}, true)
}

func countPortManagerWorkers(t *testing.T) int {
	t.Helper()
	var profile bytes.Buffer
	require.NoError(t, pprof.Lookup("goroutine").WriteTo(&profile, 2))
	count := 0
	for stack := range strings.SplitSeq(profile.String(), "\n\n") {
		if strings.Contains(stack, "github.com/fatedier/frp/server/ports.(*Manager).cleanReservedPortsWorker") {
			count++
		}
	}
	return count
}

func TestNewServiceDoesNotLeakPortManagerWorkersOnWebServerFailure(t *testing.T) {
	for _, mode := range []string{"port conflict", "invalid TLS certificate"} {
		t.Run(mode, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer ln.Close()
			cfg := &v1.ServerConfig{
				BindAddr:      "127.0.0.1",
				ProxyBindAddr: "127.0.0.1",
				WebServer:     v1.WebServerConfig{Addr: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port},
			}
			require.NoError(t, cfg.Complete())
			cfg.BindPort = 0
			if mode == "invalid TLS certificate" {
				dir := t.TempDir()
				certFile := filepath.Join(dir, "cert.pem")
				keyFile := filepath.Join(dir, "key.pem")
				require.NoError(t, os.WriteFile(certFile, []byte("invalid certificate"), 0o600))
				require.NoError(t, os.WriteFile(keyFile, []byte("invalid key"), 0o600))
				cfg.WebServer.TLS = &v1.TLSConfig{CertFile: certFile, KeyFile: keyFile}
				require.NoError(t, ln.Close())
			}

			workersBefore := countPortManagerWorkers(t)
			for range 5 {
				svr, err := NewService(cfg)
				if svr != nil {
					defer svr.Close()
				}
				require.Nil(t, svr)
				if mode == "port conflict" {
					var listenErr *net.OpError
					require.ErrorAs(t, err, &listenErr)
					require.Equal(t, "listen", listenErr.Op)
					require.Equal(t, ln.Addr().String(), listenErr.Addr.String())
				} else {
					require.ErrorContains(t, err, "failed to find any PEM data in certificate input")
				}
			}
			workersAfter := countPortManagerWorkers(t)
			t.Logf("port-manager workers after 5 failed constructions: before=%d after=%d", workersBefore, workersAfter)
			require.Equal(t, workersBefore, workersAfter, "failed construction leaked port-manager workers")
		})
	}
}

func TestNewServiceReleasesListenersOnWebServerFailure(t *testing.T) {
	for _, mode := range []string{"port conflict", "invalid TLS certificate"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &v1.ServerConfig{
				BindAddr:      "127.0.0.1",
				ProxyBindAddr: "127.0.0.1",
				WebServer:     v1.WebServerConfig{Addr: "127.0.0.1"},
			}
			require.NoError(t, cfg.Complete())
			dir := t.TempDir()
			cfg.SSHTunnelGateway.AutoGenPrivateKeyPath = filepath.Join(dir, "ssh_key")

			// Hold reservations together so each listener gets a distinct port.
			tcpPorts := []*int{
				&cfg.BindPort, &cfg.TCPMuxHTTPConnectPort, &cfg.SSHTunnelGateway.BindPort,
				&cfg.VhostHTTPPort, &cfg.VhostHTTPSPort, &cfg.WebServer.Port,
			}
			tcpReservations := make([]net.Listener, 0, len(tcpPorts))
			tcpAddresses := make([]string, 0, len(tcpPorts))
			for _, port := range tcpPorts {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { _ = ln.Close() })
				*port = ln.Addr().(*net.TCPAddr).Port
				tcpReservations = append(tcpReservations, ln)
				tcpAddresses = append(tcpAddresses, ln.Addr().String())
			}
			udpPorts := []*int{&cfg.KCPBindPort, &cfg.QUICBindPort}
			udpReservations := make([]net.PacketConn, 0, len(udpPorts))
			udpAddresses := make([]string, 0, len(udpPorts))
			for _, port := range udpPorts {
				conn, err := net.ListenPacket("udp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { _ = conn.Close() })
				*port = conn.LocalAddr().(*net.UDPAddr).Port
				udpReservations = append(udpReservations, conn)
				udpAddresses = append(udpAddresses, conn.LocalAddr().String())
			}

			webReservation := tcpReservations[len(tcpReservations)-1]
			if mode == "invalid TLS certificate" {
				certFile := filepath.Join(dir, "cert.pem")
				keyFile := filepath.Join(dir, "key.pem")
				require.NoError(t, os.WriteFile(certFile, []byte("invalid certificate"), 0o600))
				require.NoError(t, os.WriteFile(keyFile, []byte("invalid key"), 0o600))
				cfg.WebServer.TLS = &v1.TLSConfig{CertFile: certFile, KeyFile: keyFile}
				require.NoError(t, webReservation.Close())
			}
			for _, ln := range tcpReservations[:len(tcpReservations)-1] {
				require.NoError(t, ln.Close())
			}
			for _, conn := range udpReservations {
				require.NoError(t, conn.Close())
			}

			svr, err := NewService(cfg)
			if svr != nil {
				defer svr.Close()
			}
			require.Nil(t, svr)
			if mode == "port conflict" {
				var listenErr *net.OpError
				require.ErrorAs(t, err, &listenErr)
				require.Equal(t, "listen", listenErr.Op)
				require.Equal(t, webReservation.Addr().String(), listenErr.Addr.String())
				assertLifecyclePorts(t, []string{webReservation.Addr().String()}, false)
				require.NoError(t, webReservation.Close())
			} else {
				require.ErrorContains(t, err, "failed to find any PEM data in certificate input")
			}
			assertLifecyclePorts(t, tcpAddresses, true)
			for _, address := range udpAddresses {
				conn, err := net.ListenPacket("udp", address)
				require.NoError(t, err, "UDP listener %s was not released", address)
				require.NoError(t, conn.Close())
			}
		})
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

func TestMetricsCollectors(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		webPort, metricsPort    int
		enablePrometheus        bool
		wantMem, wantPrometheus bool
	}{
		{name: "nothing enabled"},
		{name: "web server only", webPort: 7500, wantMem: true},
		{
			name: "web server with enablePrometheus", webPort: 7500, enablePrometheus: true,
			wantMem: true, wantPrometheus: true,
		},
		{
			// enablePrometheus alone has no endpoint to serve metrics from, so the
			// Prometheus collector must stay off, as it did before metricsServer.
			name: "enablePrometheus without listeners", enablePrometheus: true,
		},
		{
			// The in-memory collector only backs the dashboard API.
			name: "metrics server only", metricsPort: 7501, wantPrometheus: true,
		},
		{
			name: "both servers", webPort: 7500, metricsPort: 7501,
			wantMem: true, wantPrometheus: true,
		},
		{
			name: "both servers with enablePrometheus", webPort: 7500, metricsPort: 7501, enablePrometheus: true,
			wantMem: true, wantPrometheus: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &v1.ServerConfig{
				WebServer:        v1.WebServerConfig{Port: tc.webPort},
				MetricsServer:    v1.MetricsServerConfig{Port: tc.metricsPort},
				EnablePrometheus: tc.enablePrometheus,
			}
			mem, prometheus := metricsCollectors(cfg)
			require.Equal(t, tc.wantMem, mem, "EnableMem")
			require.Equal(t, tc.wantPrometheus, prometheus, "EnablePrometheus")
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
