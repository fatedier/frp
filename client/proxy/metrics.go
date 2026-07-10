// Copyright 2025 The frp Authors
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

//go:build !frps

package proxy

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
)

// udpIdleTimeout is how long after the last real UDP packet a P2P UDP tunnel is
// still counted as an active session. A P2P UDP stream is kept alive by heartbeat
// pings, so without this it would be reported as "1 connection" forever; instead
// it drops to 0 shortly after the app stops sending UDP.
const udpIdleTimeout = 15 * time.Second

// p2pMetrics measures a provider proxy's P2P tunnel — byte counters plus a
// session count — and periodically reports the delta to frps, which cannot
// observe P2P data itself (it bypasses frps through the punched hole). Session
// counting is split by transport: TCP streams are counted precisely while open
// (libio.Join returns exactly on close), while the single persistent UDP stream
// is counted only while UDP packets are recently flowing. Only the P2P path is
// reported; the relay-fallback path is measured by frps, so the two never double
// count.
type p2pMetrics struct {
	// name is the wire proxy name (with user prefix) exactly as frps keys it.
	name string
	mt   transport.MessageTransporter

	tcpConns    atomic.Int64 // active TCP streams
	lastUDPNano atomic.Int64 // unix-nano of the last real UDP packet; 0 = none
	in          atomic.Int64 // cumulative bytes read from the tunnel
	out         atomic.Int64 // cumulative bytes written to the tunnel

	lastConns int64
	lastIn    int64
	lastOut   int64

	started atomic.Bool
}

func newP2PMetrics(name string, mt transport.MessageTransporter) *p2pMetrics {
	return &p2pMetrics{name: name, mt: mt}
}

// countBytes wraps a tunnel stream so its bytes are counted. It does NOT count a
// session — session counting is done per-transport (tcpOpen/tcpClose for TCP,
// udpActivity for UDP).
func (m *p2pMetrics) countBytes(c net.Conn) net.Conn {
	if m == nil {
		return c
	}
	return &byteCountConn{Conn: c, m: m}
}

func (m *p2pMetrics) tcpOpen() {
	if m != nil {
		m.tcpConns.Add(1)
	}
}

func (m *p2pMetrics) tcpClose() {
	if m != nil {
		m.tcpConns.Add(-1)
	}
}

// udpActivity records that a real UDP packet just flowed on the tunnel.
func (m *p2pMetrics) udpActivity() {
	if m != nil {
		m.lastUDPNano.Store(time.Now().UnixNano())
	}
}

// currentConns is TCP streams plus 1 if UDP has been active within udpIdleTimeout.
func (m *p2pMetrics) currentConns() int64 {
	c := m.tcpConns.Load()
	if last := m.lastUDPNano.Load(); last != 0 && time.Since(time.Unix(0, last)) < udpIdleTimeout {
		c++
	}
	return c
}

// startReporter launches the periodic delta reporter once (idempotent). It stops
// when ctx is done after a final flush that zeroes the reported connection count.
func (m *p2pMetrics) startReporter(ctx context.Context) {
	if m == nil || m.mt == nil {
		return
	}
	if !m.started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		// Report every 10s. flush() is a no-op when nothing moved, so idle
		// proxies stay silent and the message itself is tiny.
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				// Final report: force our reported connections to 0 so frps does
				// not keep a stale value on graceful shutdown.
				m.tcpConns.Store(0)
				m.lastUDPNano.Store(0)
				m.flush()
				return
			case <-ticker.C:
				m.flush()
			}
		}
	}()
}

// flush sends the change since the last report; it is a no-op when nothing moved.
func (m *p2pMetrics) flush() {
	conns := m.currentConns()
	in := m.in.Load()
	out := m.out.Load()
	dConns := conns - m.lastConns
	dIn := in - m.lastIn
	dOut := out - m.lastOut
	if dConns == 0 && dIn == 0 && dOut == 0 {
		return
	}
	m.lastConns, m.lastIn, m.lastOut = conns, in, out
	_ = m.mt.Send(&msg.ProxyMetrics{
		ProxyName:       m.name,
		ConnsDelta:      dConns,
		TrafficInDelta:  dIn,
		TrafficOutDelta: dOut,
	})
}

type byteCountConn struct {
	net.Conn
	m *p2pMetrics
}

func (c *byteCountConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.m.in.Add(int64(n))
	}
	return n, err
}

func (c *byteCountConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.m.out.Add(int64(n))
	}
	return n, err
}
