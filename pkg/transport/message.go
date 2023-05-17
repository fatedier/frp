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

package transport

import (
	"context"
	"reflect"
	"sync"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/msg"
)

type MessageTransporter interface {
	Send(msg.Message) error
	// Recv(ctx context.Context, laneKey string, msgType string) (Message, error)
	// Do will first send msg, then recv msg with the same laneKey and specified msgType.
	Do(ctx context.Context, req msg.Message, laneKey, recvMsgType string) (msg.Message, error)
	Dispatch(m msg.Message, laneKey string) bool
	DispatchWithType(m msg.Message, msgType, laneKey string) bool
}

func NewMessageTransporter(sendCh chan msg.Message) MessageTransporter {
	return &transporterImpl{
		sendCh:   sendCh,
		registry: make(map[string]map[string]chan msg.Message),
	}
}

type transporterImpl struct {
	sendCh chan msg.Message

	// First key is message type and second key is lane key.
	// Dispatch will dispatch message to releated channel by its message type
	// and lane key.
	registry map[string]map[string]chan msg.Message
	mu       sync.RWMutex
}

func (impl *transporterImpl) Send(m msg.Message) error {
	return errors.PanicToError(func() {
		impl.sendCh <- m
	})
}

func (impl *transporterImpl) Do(ctx context.Context, req msg.Message, laneKey, recvMsgType string) (msg.Message, error) {
	ch := make(chan msg.Message, 1)
	defer close(ch)
	unregisterFn := impl.registerMsgChan(ch, laneKey, recvMsgType)
	defer unregisterFn()

	if err := impl.Send(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func (impl *transporterImpl) DispatchWithType(m msg.Message, msgType, laneKey string) bool {
	var ch chan msg.Message
	impl.mu.RLock()
	byLaneKey, ok := impl.registry[msgType]
	if ok {
		ch = byLaneKey[laneKey]
	}
	impl.mu.RUnlock()

	if ch == nil {
		return false
	}

	if err := errors.PanicToError(func() {
		ch <- m
	}); err != nil {
		return false
	}
	return true
}

func (impl *transporterImpl) Dispatch(m msg.Message, laneKey string) bool {
	msgType := reflect.TypeOf(m).Elem().Name()
	return impl.DispatchWithType(m, msgType, laneKey)
}

func (impl *transporterImpl) registerMsgChan(recvCh chan msg.Message, laneKey string, msgType string) (unregister func()) {
	impl.mu.Lock()
	byLaneKey, ok := impl.registry[msgType]
	if !ok {
		byLaneKey = make(map[string]chan msg.Message)
		impl.registry[msgType] = byLaneKey
	}
	byLaneKey[laneKey] = recvCh
	impl.mu.Unlock()

	unregister = func() {
		impl.mu.Lock()
		delete(byLaneKey, laneKey)
		impl.mu.Unlock()
	}
	return
}
