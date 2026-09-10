package client

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/client/visitor"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type gracefulCloseTestConnector struct {
	conn net.Conn
}

func (*gracefulCloseTestConnector) Connect() (*msg.Conn, error) { return nil, net.ErrClosed }
func (c *gracefulCloseTestConnector) Close() error              { return c.conn.Close() }

func newGracefulCloseTestService() *Service {
	ctx := context.Background()
	common := &v1.ClientCommonConfig{}
	serverConn, clientConn := net.Pipe()
	ctl := &Control{
		ctx: ctx,
		sessionCtx: &SessionContext{
			Common:    common,
			RunID:     "graceful-close-race",
			Conn:      msg.NewConn(clientConn, msg.NewV1ReadWriter(clientConn)),
			Connector: &gracefulCloseTestConnector{conn: serverConn},
		},
		doneCh: make(chan struct{}),
	}
	ctl.pm = proxy.NewManager(ctx, common, nil, nil, nil, "")
	ctl.vm = visitor.NewManager(ctx, "graceful-close-race", common, nil, nil, nil, "")
	return &Service{ctl: ctl, cancel: context.CancelCauseFunc(func(error) {})}
}

func TestGracefulCloseAndStopSynchronizeDuration(t *testing.T) {
	for i := range 10000 {
		svr := newGracefulCloseTestService()
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			svr.GracefulClose(time.Duration(i))
		}()
		go func() {
			defer wg.Done()
			<-start
			svr.stop()
		}()
		close(start)
		wg.Wait()
	}
}

func TestGracefulCloseDoesNotBlockDuringStop(t *testing.T) {
	const gracefulDuration = 200 * time.Millisecond

	svr := newGracefulCloseTestService()
	svr.GracefulClose(gracefulDuration)
	stopDone := make(chan struct{})
	go func() {
		svr.stop()
		close(stopDone)
	}()
	defer func() {
		select {
		case <-stopDone:
		case <-time.After(time.Second):
			t.Error("stop did not finish")
		}
	}()

	deadline := time.Now().Add(time.Second)
	for svr.ctlMu.TryLock() {
		svr.ctlMu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("stop did not acquire ctlMu")
		}
		time.Sleep(time.Millisecond)
	}

	start := time.Now()
	svr.GracefulClose(0)
	if elapsed := time.Since(start); elapsed >= gracefulDuration/2 {
		t.Fatalf("GracefulClose blocked for %v while stop was waiting", elapsed)
	}
}

// TestUpdateAllConfigurerAndStopSynchronizeControl verifies that configuration
// updates and shutdown synchronize access to the active control.
func TestUpdateAllConfigurerAndStopSynchronizeControl(t *testing.T) {
	for range 1000 {
		svr := newGracefulCloseTestService()
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			<-start
			_ = svr.UpdateAllConfigurer(nil, nil)
		}()
		go func() {
			defer wg.Done()
			<-start
			svr.stop()
		}()

		close(start)
		wg.Wait()
	}
}

// TestKeepControllerWorkingWaitsForCapturedControl verifies that replacing the
// Service field does not redirect a monitor away from its original control.
func TestKeepControllerWorkingWaitsForCapturedControl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	initialControl := &Control{doneCh: make(chan struct{})}
	currentControl := &Control{doneCh: make(chan struct{})}
	svr := &Service{ctx: ctx, ctl: currentControl}
	monitorDone := make(chan struct{})
	go func() {
		svr.keepControllerWorking(initialControl)
		close(monitorDone)
	}()

	close(initialControl.doneCh)
	select {
	case <-monitorDone:
	case <-time.After(time.Second):
		t.Fatal("controller monitor did not finish after the captured control closed")
	}
}
