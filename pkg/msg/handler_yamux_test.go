// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package msg

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fmux "github.com/fatedier/yamux"

	"github.com/fatedier/frp/pkg/proto/wire"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

// Delay one real TCP write without injecting an error. The stream's write
// failure must come from yamux's ConnectionWriteTimeout.
type delayedTCPConn struct {
	net.Conn
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (c *delayedTCPConn) Write(p []byte) (int, error) {
	if c.armed.Swap(false) {
		close(c.entered)
		<-c.release
	}
	return c.Conn.Write(p)
}

type observedYamuxConn struct {
	net.Conn
	writes      chan error
	readEntered chan struct{}
	readOnce    sync.Once
	closed      chan struct{}
	closeCalls  atomic.Int32
}

func (c *observedYamuxConn) Read(p []byte) (int, error) {
	c.readOnce.Do(func() { close(c.readEntered) })
	return c.Conn.Read(p)
}

func (c *observedYamuxConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.writes <- err
	return n, err
}

func (c *observedYamuxConn) Close() error {
	c.closeCalls.Add(1)
	err := c.Conn.Close()
	close(c.closed)
	return err
}

func newLoopbackYamuxPair(t *testing.T) (*fmux.Session, *fmux.Session, *delayedTCPConn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	clientTCP, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clientTCP.Close() })
	serverTCP, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverTCP.Close() })
	gate := &delayedTCPConn{Conn: clientTCP, entered: make(chan struct{}), release: make(chan struct{})}
	config := fmux.DefaultConfig()
	config.ConnectionWriteTimeout = 80 * time.Millisecond
	config.LogOutput = io.Discard
	client, err := fmux.Client(gate, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server, err := fmux.Server(serverTCP, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	return client, server, gate
}

func loopbackCryptoConn(t *testing.T, conn net.Conn, protocol string, role netpkg.AEADCryptoRole) *Conn {
	t.Helper()
	var rw io.ReadWriter
	var err error
	if protocol == wire.ProtocolV2 {
		rw, err = netpkg.NewAEADCryptoReadWriter(
			conn, regressionKey, role, wire.AEADAlgorithmAES256GCM, []byte("isolated-test-transcript"))
	} else {
		rw, err = netpkg.NewCryptoReadWriter(conn, regressionKey)
	}
	if err != nil {
		t.Fatal(err)
	}
	return NewConn(conn, NewReadWriter(rw, protocol))
}

func loopbackControlPair(t *testing.T, client, server *fmux.Session, protocol string) (*Conn, *Conn, *observedYamuxConn) {
	t.Helper()
	local, err := client.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	remote, err := server.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	observed := &observedYamuxConn{
		Conn: local, writes: make(chan error, 16), readEntered: make(chan struct{}), closed: make(chan struct{}),
	}
	return loopbackCryptoConn(t, observed, protocol, netpkg.AEADCryptoRoleClient),
		loopbackCryptoConn(t, remote, protocol, netpkg.AEADCryptoRoleServer), observed
}

func loopbackReadReq(t *testing.T, conn *Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	defer conn.SetReadDeadline(time.Time{})
	m, err := conn.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(*ReqWorkConn); !ok {
		t.Fatalf("decoded %T, want ReqWorkConn", m)
	}
}

func TestDispatcherYamuxWriteTimeoutInterruptsRead(t *testing.T) {
	for _, protocol := range []string{wire.ProtocolV1, wire.ProtocolV2} {
		t.Run(protocol, func(t *testing.T) {
			client, server, gate := newLoopbackYamuxPair(t)
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(gate.release) })
			control, peer, observed := loopbackControlPair(t, client, server, protocol)
			if err := control.WriteMsg(&ReqWorkConn{}); err != nil {
				t.Fatal(err)
			}
			loopbackReadReq(t, peer)
			// Warm-up has completed, including all crypto header writes.
			for len(observed.writes) > 0 {
				<-observed.writes
			}
			d := NewDispatcher(control)
			readExited := make(chan struct{})
			go func() {
				defer close(readExited)
				d.readLoop()
			}()
			regressionAwait(t, observed.readEntered, "control reader entry")
			gate.armed.Store(true)
			// Queue before starting the writer so this Send cannot race its failure.
			if err := d.Send(&ReqWorkConn{}); err != nil {
				t.Fatal(err)
			}
			go d.sendLoop()
			regressionAwait(t, gate.entered, "TCP write delay barrier")
			select {
			case err := <-observed.writes:
				if !errors.Is(err, fmux.ErrConnectionWriteTimeout) {
					t.Fatalf("stream write error: %v, want yamux connection write timeout", err)
				}
				t.Logf("production yamux write failed: %v", err)
			case <-time.After(time.Second):
				t.Fatal("yamux write did not time out")
			}
			regressionAwait(t, d.stopCh, "write failure stops sends")
			if err := d.Send(&ReqWorkConn{}); !errors.Is(err, io.EOF) {
				t.Fatalf("Send after stop: %v, want EOF", err)
			}
			releaseOnce.Do(func() { close(gate.release) })
			regressionAwait(t, observed.closed, "control Close returns")
			// peer never closes its write half: local yamux Close alone cannot
			// interrupt Read. Neither a peer FIN nor session cleanup may help.
			regressionAwait(t, d.Done(), "Done without peer FIN")
			regressionAwait(t, readExited, "control readLoop exit without peer FIN")
			if observed.closeCalls.Load() != 1 {
				t.Fatalf("control closed %d times, want once", observed.closeCalls.Load())
			}
			if client.IsClosed() || server.IsClosed() {
				t.Fatal("yamux session unexpectedly closed")
			}
			fresh, freshPeer, freshObserved := loopbackControlPair(t, client, server, protocol)
			for range 5 {
				if err := fresh.WriteMsg(&ReqWorkConn{}); err != nil {
					t.Fatal(err)
				}
				loopbackReadReq(t, freshPeer)
				for len(freshObserved.writes) > 0 {
					<-freshObserved.writes
				}
			}
			t.Log("Done/readLoop exited without peer FIN; Close once; same yamux session decrypted five fresh messages")
		})
	}
}
