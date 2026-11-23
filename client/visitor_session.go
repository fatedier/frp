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

package client

import (
	"context"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/xlog"
)

// visitorSessions backs transport.visitorSession = "dedicated": every visitor dials its own session to frps (its own
// TCP connection with its own yamux session, or its own QUIC connection) instead of opening streams on the session
// shared with the control connection. Each visitor then has its own congestion window, and a lost or queued packet on
// one visitor's connection no longer stalls the streams of every other visitor in the same frpc.
//
// frps needs no change for this: it admits a visitor connection by the run id it carries (RegisterVisitorConn), not by
// the connection it arrived on.
type visitorSessions struct {
	ctx          context.Context
	cfg          *v1.ClientCommonConfig
	wireProtocol string

	mu       sync.Mutex
	sessions map[string]*dedicatedSession
}

type dedicatedSession struct {
	name string

	mu        sync.Mutex
	connector Connector
}

func newVisitorSessions(ctx context.Context, cfg *v1.ClientCommonConfig) *visitorSessions {
	if port := cfg.Transport.VisitorSessionQUICPort; port > 0 {
		// Login and proxies keep the configured transport; only the visitors' sessions use QUIC. A network that
		// blocks UDP then still logs in, and the visitor reports the failed QUIC session in its own log line.
		c := *cfg
		c.Transport.Protocol = "quic"
		c.ServerPort = port
		cfg = &c
	}
	return &visitorSessions{
		ctx:          ctx,
		cfg:          cfg,
		wireProtocol: cfg.Transport.WireProtocol,
		sessions:     make(map[string]*dedicatedSession),
	}
}

// ConnectServerFn returns the visitor's connect function. The session is opened right away, as frpc's own control
// session is at startup, so the visitor's first connection does not pay for the handshake.
func (s *visitorSessions) ConnectServerFn(name string) func() (*msg.Conn, error) {
	s.mu.Lock()
	ds := s.sessions[name]
	if ds == nil {
		ds = &dedicatedSession{name: name}
		s.sessions[name] = ds
	}
	s.mu.Unlock()

	go func() { _, _ = ds.open(s) }()
	return func() (*msg.Conn, error) {
		return ds.connect(s)
	}
}

func (ds *dedicatedSession) open(s *visitorSessions) (Connector, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.connector != nil {
		return ds.connector, nil
	}
	xl := xlog.FromContextSafe(s.ctx).Spawn().AppendPrefix(ds.name)
	c := NewConnector(s.ctx, s.cfg)
	if err := c.Open(); err != nil {
		xl.Warnf("open visitor session over %s to server error: %v", s.cfg.Transport.Protocol, err)
		return nil, err
	}
	// frps closes a connection whose first byte does not arrive within its read timeout, and a yamux client sends
	// nothing until a stream is opened. Ping once so that a session opened ahead of use is not cut before its first
	// stream; yamux keepalive takes over afterwards. QUIC sessions already exchanged their handshake.
	if impl, ok := c.(*defaultConnectorImpl); ok && impl.muxSession != nil {
		if _, err := impl.muxSession.Ping(); err != nil {
			_ = c.Close()
			xl.Warnf("open visitor session over %s to server error: %v", s.cfg.Transport.Protocol, err)
			return nil, err
		}
	}
	ds.connector = c
	xl.Infof("open visitor session over %s to server success", s.cfg.Transport.Protocol)
	return c, nil
}

// connect opens a stream on the visitor's session. A session that fails to open a stream is closed and dialed
// again once.
func (ds *dedicatedSession) connect(s *visitorSessions) (*msg.Conn, error) {
	var lastErr error
	for range 2 {
		c, err := ds.open(s)
		if err != nil {
			return nil, err
		}
		conn, err := newMessageConnector(c, s.wireProtocol).Connect()
		if err == nil {
			return conn, nil
		}
		lastErr = err
		ds.mu.Lock()
		if ds.connector == c {
			_ = c.Close()
			ds.connector = nil
		}
		ds.mu.Unlock()
	}
	return nil, lastErr
}

func (ds *dedicatedSession) close() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if ds.connector != nil {
		_ = ds.connector.Close()
		ds.connector = nil
	}
}

// Close closes the session of a visitor that was removed.
func (s *visitorSessions) Close(name string) {
	s.mu.Lock()
	ds := s.sessions[name]
	delete(s.sessions, name)
	s.mu.Unlock()
	if ds != nil {
		ds.close()
	}
}

// CloseAll closes every visitor session; the control is going away.
func (s *visitorSessions) CloseAll() {
	s.mu.Lock()
	all := s.sessions
	s.sessions = make(map[string]*dedicatedSession)
	s.mu.Unlock()
	for _, ds := range all {
		ds.close()
	}
}
