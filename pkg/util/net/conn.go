// Copyright 2016 fatedier, fatedier@gmail.com
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

package net

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	libcrypto "github.com/fatedier/golib/crypto"
	quic "github.com/quic-go/quic-go"
	"golang.org/x/crypto/hkdf"

	"github.com/fatedier/frp/pkg/util/xlog"
)

type ContextGetter interface {
	Context() context.Context
}

type ContextSetter interface {
	WithContext(ctx context.Context)
}

func NewLogFromConn(conn net.Conn) *xlog.Logger {
	if c, ok := conn.(ContextGetter); ok {
		return xlog.FromContextSafe(c.Context())
	}
	return xlog.New()
}

func NewContextFromConn(conn net.Conn) context.Context {
	if c, ok := conn.(ContextGetter); ok {
		return c.Context()
	}
	return context.Background()
}

// ContextConn is the connection with context
type ContextConn struct {
	net.Conn

	ctx context.Context
}

func NewContextConn(ctx context.Context, c net.Conn) *ContextConn {
	return &ContextConn{
		Conn: c,
		ctx:  ctx,
	}
}

func (c *ContextConn) WithContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *ContextConn) Context() context.Context {
	return c.ctx
}

type WrapReadWriteCloserConn struct {
	io.ReadWriteCloser

	underConn net.Conn

	remoteAddr net.Addr
}

func WrapReadWriteCloserToConn(rwc io.ReadWriteCloser, underConn net.Conn) *WrapReadWriteCloserConn {
	return &WrapReadWriteCloserConn{
		ReadWriteCloser: rwc,
		underConn:       underConn,
	}
}

func (conn *WrapReadWriteCloserConn) LocalAddr() net.Addr {
	if conn.underConn != nil {
		return conn.underConn.LocalAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) SetRemoteAddr(addr net.Addr) {
	conn.remoteAddr = addr
}

func (conn *WrapReadWriteCloserConn) RemoteAddr() net.Addr {
	if conn.remoteAddr != nil {
		return conn.remoteAddr
	}
	if conn.underConn != nil {
		return conn.underConn.RemoteAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *WrapReadWriteCloserConn) SetDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (conn *WrapReadWriteCloserConn) SetReadDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetReadDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (conn *WrapReadWriteCloserConn) SetWriteDeadline(t time.Time) error {
	if conn.underConn != nil {
		return conn.underConn.SetWriteDeadline(t)
	}
	return &net.OpError{Op: "set", Net: "wrap", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

type CloseNotifyConn struct {
	net.Conn

	// 1 means closed
	closeFlag atomic.Int32

	closeFn func(error)
}

// closeFn will be only called once with the error (nil if Close() was called, non-nil if CloseWithError() was called)
func WrapCloseNotifyConn(c net.Conn, closeFn func(error)) *CloseNotifyConn {
	return &CloseNotifyConn{
		Conn:    c,
		closeFn: closeFn,
	}
}

func (cc *CloseNotifyConn) Close() (err error) {
	pflag := cc.closeFlag.Swap(1)
	if pflag == 0 {
		err = cc.Conn.Close()
		if cc.closeFn != nil {
			cc.closeFn(nil)
		}
	}
	return
}

// CloseWithError closes the connection and passes the error to the close callback.
func (cc *CloseNotifyConn) CloseWithError(err error) error {
	pflag := cc.closeFlag.Swap(1)
	if pflag == 0 {
		closeErr := cc.Conn.Close()
		if cc.closeFn != nil {
			cc.closeFn(err)
		}
		return closeErr
	}
	return nil
}

type StatsConn struct {
	net.Conn

	closed     atomic.Int64 // 1 means closed
	totalRead  int64
	totalWrite int64
	statsFunc  func(totalRead, totalWrite int64)
}

func WrapStatsConn(conn net.Conn, statsFunc func(total, totalWrite int64)) *StatsConn {
	return &StatsConn{
		Conn:      conn,
		statsFunc: statsFunc,
	}
}

func (statsConn *StatsConn) Read(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Read(p)
	statsConn.totalRead += int64(n)
	return
}

func (statsConn *StatsConn) Write(p []byte) (n int, err error) {
	n, err = statsConn.Conn.Write(p)
	statsConn.totalWrite += int64(n)
	return
}

func (statsConn *StatsConn) Close() (err error) {
	old := statsConn.closed.Swap(1)
	if old != 1 {
		err = statsConn.Conn.Close()
		if statsConn.statsFunc != nil {
			statsConn.statsFunc(statsConn.totalRead, statsConn.totalWrite)
		}
	}
	return
}

type wrapQuicStream struct {
	*quic.Stream
	c *quic.Conn
}

func QuicStreamToNetConn(s *quic.Stream, c *quic.Conn) net.Conn {
	return &wrapQuicStream{
		Stream: s,
		c:      c,
	}
}

func (conn *wrapQuicStream) LocalAddr() net.Addr {
	if conn.c != nil {
		return conn.c.LocalAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *wrapQuicStream) RemoteAddr() net.Addr {
	if conn.c != nil {
		return conn.c.RemoteAddr()
	}
	return (*net.TCPAddr)(nil)
}

func (conn *wrapQuicStream) Close() error {
	conn.CancelRead(0)
	return conn.Stream.Close()
}

func NewCryptoReadWriter(rw io.ReadWriter, key []byte) (io.ReadWriter, error) {
	encReader := libcrypto.NewReader(rw, key)
	encWriter, err := libcrypto.NewWriter(rw, key)
	if err != nil {
		return nil, err
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: encReader,
		Writer: encWriter,
	}, nil
}

type AEADCryptoRole int

const (
	AEADCryptoRoleClient AEADCryptoRole = iota + 1
	AEADCryptoRoleServer
)

const (
	aeadControlHKDFInfoPrefix   = "frp wire v2 control aead"
	aeadDirectionClientToServer = "client-to-server"
	aeadDirectionServerToClient = "server-to-client"
)

// NewAEADCryptoReadWriter wraps rw with framed AEAD encryption for the v2
// control channel. Frames and their order are authenticated, but end-of-stream
// is not: a clean EOF at a frame boundary is returned as normal EOF by the
// underlying AEAD stream. Protocols that need truncation detection for finite
// objects must add their own authenticated final message.
func NewAEADCryptoReadWriter(
	rw io.ReadWriter,
	key []byte,
	role AEADCryptoRole,
	algorithm string,
	transcriptHash []byte,
) (io.ReadWriter, error) {
	clientToServerKey, serverToClientKey, err := deriveAEADControlKeys(key, algorithm, transcriptHash)
	if err != nil {
		return nil, err
	}

	var readKey, writeKey []byte
	switch role {
	case AEADCryptoRoleClient:
		readKey = serverToClientKey
		writeKey = clientToServerKey
	case AEADCryptoRoleServer:
		readKey = clientToServerKey
		writeKey = serverToClientKey
	default:
		return nil, errors.New("invalid aead crypto role")
	}

	encReader, err := libcrypto.NewAEADStreamReader(rw, libcrypto.AEADStreamOptions{
		Algorithm: libcrypto.AEADAlgorithm(algorithm),
		Key:       readKey,
	})
	if err != nil {
		return nil, err
	}
	encWriter, err := libcrypto.NewAEADStreamWriter(rw, libcrypto.AEADStreamOptions{
		Algorithm: libcrypto.AEADAlgorithm(algorithm),
		Key:       writeKey,
	})
	if err != nil {
		return nil, err
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: encReader,
		Writer: encWriter,
	}, nil
}

func deriveAEADControlKeys(key []byte, algorithm string, transcriptHash []byte) (clientToServerKey, serverToClientKey []byte, err error) {
	clientToServerKey, err = deriveAEADControlKey(key, algorithm, transcriptHash, aeadDirectionClientToServer)
	if err != nil {
		return nil, nil, err
	}
	serverToClientKey, err = deriveAEADControlKey(key, algorithm, transcriptHash, aeadDirectionServerToClient)
	if err != nil {
		return nil, nil, err
	}
	return clientToServerKey, serverToClientKey, nil
}

func deriveAEADControlKey(key []byte, algorithm string, transcriptHash []byte, direction string) ([]byte, error) {
	info := []byte(aeadControlHKDFInfoPrefix + " " + algorithm + " " + direction)
	reader := hkdf.New(sha256.New, key, transcriptHash, info)
	out := make([]byte, libcrypto.AEADKeySize)
	if _, err := io.ReadFull(reader, out); err != nil {
		return nil, err
	}
	return out, nil
}
