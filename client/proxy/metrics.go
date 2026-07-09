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

// p2pMetrics accumulates session count + byte counters for a provider proxy's
// P2P tunnel and periodically reports the delta to frps. frps cannot observe P2P
// data itself (it flows straight through the punched hole), so without this the
// dashboard shows 0 traffic / 0 connections for xtcp/xudp/xtcp+xudp. Only the
// P2P path is tracked — the relay-fallback path is measured by frps directly, so
// the two sum without double counting.
type p2pMetrics struct {
	// name is the wire proxy name (with user prefix) exactly as frps keys it.
	name string
	mt   transport.MessageTransporter

	conns atomic.Int64 // current active tunnel streams
	in    atomic.Int64 // cumulative bytes read from the tunnel (peer -> local)
	out   atomic.Int64 // cumulative bytes written to the tunnel (local -> peer)

	lastConns int64
	lastIn    int64
	lastOut   int64

	started atomic.Bool
}

func newP2PMetrics(name string, mt transport.MessageTransporter) *p2pMetrics {
	return &p2pMetrics{name: name, mt: mt}
}

// track wraps a tunnel stream so its bytes are counted, and marks one more active
// session; the session is released when the returned conn is closed.
func (m *p2pMetrics) track(c net.Conn) net.Conn {
	if m == nil {
		return c
	}
	m.conns.Add(1)
	return &countingConn{Conn: c, m: m}
}

// startReporter launches the periodic delta reporter once (idempotent). It stops
// when ctx is done, sending one final flush.
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
	conns := m.conns.Load()
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

type countingConn struct {
	net.Conn
	m      *p2pMetrics
	closed atomic.Bool
}

func (c *countingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.m.in.Add(int64(n))
	}
	return n, err
}

func (c *countingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.m.out.Add(int64(n))
	}
	return n, err
}

func (c *countingConn) Close() error {
	if c.closed.CompareAndSwap(false, true) {
		c.m.conns.Add(-1)
	}
	return c.Conn.Close()
}
