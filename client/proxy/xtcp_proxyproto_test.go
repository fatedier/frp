package proxy

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	pp "github.com/pires/go-proxyproto"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/xlog"
)

func TestXTCPProxy_InsertsProxyProtocolV1(t *testing.T) {
	// Start backend that verifies it receives a PROXY v1 header, then the payload
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen backend: %v", err)
	}
	defer ln.Close()

	backendAddr := ln.Addr().(*net.TCPAddr)
	payload := []byte("HELLO")
	backendDone := make(chan struct{})
	go func() {
		defer close(backendDone)
		c, err := ln.Accept()
		if err != nil {
			t.Errorf("backend accept: %v", err)
			return
		}
		defer c.Close()
		// Decode PROXY header
		br := bufio.NewReader(c)
		h, err := pp.Read(br)
		if err != nil {
			t.Errorf("read proxy header: %v", err)
			return
		}
		if h.SourceAddr == nil {
			t.Errorf("nil source addr in proxy header")
		}
		// Then read application payload using the same reader
		got := make([]byte, len(payload))
		if _, err := br.Read(got); err != nil {
			t.Errorf("read payload: %v", err)
			return
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("payload mismatch: %q != %q", got, payload)
		}
	}()

	// Prepare BaseProxy with XTCP type and PROXY v1 enabled
	baseCfg := &v1.ProxyBaseConfig{
		Name: "test-xtcp",
		Type: string(v1.ProxyTypeXTCP),
		Transport: v1.ProxyTransport{
			ProxyProtocolVersion: "v1",
		},
		ProxyBackend: v1.ProxyBackend{
			LocalIP:   backendAddr.IP.String(),
			LocalPort: backendAddr.Port,
		},
	}
	pxy := &BaseProxy{
		baseCfg: baseCfg,
		ctx:     context.Background(),
		xl:      xlog.New(),
	}

	// Create a pipe as the workConn; write XTCP meta then payload
	r1, r2 := net.Pipe()
	defer r1.Close()
	defer r2.Close()

	go func() {
		// Write XTCP meta header for client source 198.51.100.7:40000, then payload
		_ = netpkg.WriteXTCPClientMeta(r2, &net.TCPAddr{IP: net.ParseIP("198.51.100.7"), Port: 40000})
		r2.Write(payload)
		// allow some time then close
		time.Sleep(50 * time.Millisecond)
		r2.Close()
	}()

	m := &v1.ProxyBaseConfig{} // not used here beyond defaults
	_ = m // silence unused in this scope

	// Start handler (will dial backend and join streams)
	startMsg := &struct{ v interface{} }{}
	_ = startMsg // placeholder to keep signature parity in mind

	pxy.HandleTCPWorkConnection(r1, &msg.StartWorkConn{}, nil)

	select {
	case <-backendDone:
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not complete in time")
	}
}