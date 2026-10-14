// Copyright 2023 The frp Authors
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
	"context"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/proto/wire"
)

type ReadWriter interface {
	ReadMsg() (Message, error)
	ReadMsgInto(Message) error
	WriteMsg(Message) error
}

type Conn struct {
	net.Conn
	rw ReadWriter
}

func NewConn(conn net.Conn, rw ReadWriter) *Conn {
	return &Conn{
		Conn: conn,
		rw:   rw,
	}
}

func (c *Conn) ReadMsg() (Message, error) {
	return c.rw.ReadMsg()
}

func (c *Conn) ReadMsgInto(m Message) error {
	return c.rw.ReadMsgInto(m)
}

func (c *Conn) WriteMsg(m Message) error {
	return c.rw.WriteMsg(m)
}

func (c *Conn) Context() context.Context {
	if getter, ok := c.Conn.(interface{ Context() context.Context }); ok {
		return getter.Context()
	}
	return context.Background()
}

func (c *Conn) WithContext(ctx context.Context) {
	if setter, ok := c.Conn.(interface{ WithContext(context.Context) }); ok {
		setter.WithContext(ctx)
	}
}

type V1ReadWriter struct {
	rw io.ReadWriter
}

func NewV1ReadWriter(rw io.ReadWriter) ReadWriter {
	return &V1ReadWriter{rw: rw}
}

// NewReadWriter wraps rw with the message codec for the selected wire protocol.
// An empty protocol keeps the historical v1 behavior for tests and older call sites.
func NewReadWriter(rw io.ReadWriter, wireProtocol string) ReadWriter {
	switch wireProtocol {
	case wire.ProtocolV2:
		return NewV2ReadWriter(rw)
	case "", wire.ProtocolV1:
		return NewV1ReadWriter(rw)
	default:
		return NewV1ReadWriter(rw)
	}
}

func (rw *V1ReadWriter) ReadMsg() (Message, error) {
	return ReadMsg(rw.rw)
}

func (rw *V1ReadWriter) ReadMsgInto(m Message) error {
	return ReadMsgInto(rw.rw, m)
}

func (rw *V1ReadWriter) WriteMsg(m Message) error {
	return WriteMsg(rw.rw, m)
}

func AsyncHandler(f func(Message)) func(Message) {
	return func(m Message) {
		go f(m)
	}
}

// Dispatcher is used to send messages to net.Conn or register handlers for messages read from net.Conn.
type Dispatcher struct {
	rw ReadWriter

	sendCh      chan Message
	doneCh      chan struct{}
	stopCh      chan struct{}
	stopOnce    sync.Once
	msgHandlers map[reflect.Type]func(Message)
}

// NewDispatcher uses SetReadDeadline, when supported, and Close, when rw
// implements io.Closer, to interrupt ReadMsg after a write failure. Without a
// working read deadline, Close or the caller must arrange for ReadMsg to return.
// These methods must return in bounded time and must not wait for Done: Done
// closes only after ReadMsg, any synchronous handler, and stop have returned.
func NewDispatcher(rw ReadWriter) *Dispatcher {
	return &Dispatcher{
		rw:          rw,
		sendCh:      make(chan Message, 100),
		doneCh:      make(chan struct{}),
		stopCh:      make(chan struct{}),
		msgHandlers: make(map[reflect.Type]func(Message)),
	}
}

// Run will block until io.EOF or some error occurs.
func (d *Dispatcher) Run() {
	go d.sendLoop()
	go d.readLoop()
}

// stop rejects new sends and interrupts ReadMsg when supported by rw. Done remains
// owned by readLoop so the control owner cannot clean up while a handler is running.
func (d *Dispatcher) stop() {
	d.stopOnce.Do(func() {
		close(d.stopCh)
		if conn, ok := d.rw.(interface{ SetReadDeadline(time.Time) error }); ok {
			_ = conn.SetReadDeadline(time.Now())
		}
		if closer, ok := d.rw.(io.Closer); ok {
			_ = closer.Close()
		}
	})
}

func (d *Dispatcher) sendLoop() {
	for {
		select {
		case <-d.stopCh:
			return
		case m := <-d.sendCh:
			if err := d.rw.WriteMsg(m); err != nil {
				d.stop()
				return
			}
		}
	}
}

func (d *Dispatcher) readLoop() {
	defer close(d.doneCh)
	defer d.stop()
	for {
		m, err := d.rw.ReadMsg()
		if err != nil {
			return
		}

		select {
		case <-d.stopCh:
			return
		default:
		}

		if handler, ok := d.msgHandlers[reflect.TypeOf(m)]; ok {
			handler(m)
		}
	}
}

// Send queues m for writing. A nil error does not guarantee delivery.
func (d *Dispatcher) Send(m Message) error {
	select {
	case <-d.stopCh:
		return io.EOF
	default:
	}
	return d.enqueue(m)
}

func (d *Dispatcher) enqueue(m Message) error {
	select {
	case <-d.stopCh:
		return io.EOF
	case d.sendCh <- m:
		// Both cases can be ready if shutdown races with enqueueing. Do not
		// report success when shutdown is already observable after the send.
		select {
		case <-d.stopCh:
			return io.EOF
		default:
			return nil
		}
	}
}

func (d *Dispatcher) RegisterHandler(msg Message, handler func(Message)) {
	d.msgHandlers[reflect.TypeOf(msg)] = handler
}

func (d *Dispatcher) Done() chan struct{} {
	return d.doneCh
}
