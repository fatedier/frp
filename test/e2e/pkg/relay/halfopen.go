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

package relay

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// HalfOpen forwards TCP connections until Blackhole is called. A blackholed
// pair stops forwarding but deliberately retains the upstream socket so the
// peer sees a real half-open connection until the relay is closed.
type HalfOpen struct {
	bindAddr     string
	bindPort     int
	upstreamAddr string

	listener net.Listener
	done     chan struct{}
	accepted chan struct{}

	mu    sync.Mutex
	pairs []*connectionPair

	wg        sync.WaitGroup
	closeOnce sync.Once
}

type connectionPair struct {
	downstream net.Conn
	upstream   net.Conn

	mu         sync.Mutex
	blackholed bool
}

func New(upstreamAddr string) *HalfOpen {
	return &HalfOpen{
		bindAddr:     "127.0.0.1",
		upstreamAddr: upstreamAddr,
		done:         make(chan struct{}),
		accepted:     make(chan struct{}, 1),
	}
}

func (r *HalfOpen) Run() error {
	listener, err := net.Listen("tcp", net.JoinHostPort(r.bindAddr, strconv.Itoa(r.bindPort)))
	if err != nil {
		return err
	}
	r.listener = listener
	r.bindPort = listener.Addr().(*net.TCPAddr).Port

	r.wg.Add(1)
	go r.acceptLoop()
	return nil
}

func (r *HalfOpen) acceptLoop() {
	defer r.wg.Done()
	for {
		downstream, err := r.listener.Accept()
		if err != nil {
			return
		}
		upstream, err := net.DialTimeout("tcp", r.upstreamAddr, 3*time.Second)
		if err != nil {
			_ = downstream.Close()
			continue
		}

		pair := &connectionPair{downstream: downstream, upstream: upstream}
		r.mu.Lock()
		r.pairs = append(r.pairs, pair)
		r.mu.Unlock()
		select {
		case r.accepted <- struct{}{}:
		default:
		}

		r.wg.Add(1)
		go r.servePair(pair)
	}
}

func (r *HalfOpen) servePair(pair *connectionPair) {
	defer r.wg.Done()
	copyDone := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(pair.upstream, pair.downstream)
		copyDone <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(pair.downstream, pair.upstream)
		copyDone <- struct{}{}
	}()

	completed := 0
	select {
	case <-copyDone:
		completed = 1
		if !pair.isBlackholed() {
			_ = pair.downstream.Close()
			_ = pair.upstream.Close()
		}
	case <-r.done:
		_ = pair.downstream.Close()
		_ = pair.upstream.Close()
	}
	for completed < 2 {
		<-copyDone
		completed++
	}

	if pair.isBlackholed() {
		<-r.done
		_ = pair.downstream.Close()
		_ = pair.upstream.Close()
	}
}

func (r *HalfOpen) WaitForConnections(count int, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		r.mu.Lock()
		accepted := len(r.pairs)
		r.mu.Unlock()
		if accepted >= count {
			return nil
		}
		select {
		case <-r.accepted:
		case <-r.done:
			return fmt.Errorf("relay closed after accepting %d of %d connections", accepted, count)
		case <-timer.C:
			return fmt.Errorf("timed out after accepting %d of %d connections", accepted, count)
		}
	}
}

// Blackhole uses a one-based connection index in accept order.
func (r *HalfOpen) Blackhole(index int) error {
	r.mu.Lock()
	if index <= 0 || index > len(r.pairs) {
		accepted := len(r.pairs)
		r.mu.Unlock()
		return fmt.Errorf("connection %d is unavailable; accepted %d", index, accepted)
	}
	pair := r.pairs[index-1]
	r.mu.Unlock()

	pair.mu.Lock()
	if pair.blackholed {
		pair.mu.Unlock()
		return nil
	}
	pair.blackholed = true
	pair.mu.Unlock()

	now := time.Now()
	_ = pair.downstream.SetDeadline(now)
	_ = pair.upstream.SetDeadline(now)
	return nil
}

func (r *HalfOpen) Close() error {
	r.closeOnce.Do(func() {
		close(r.done)
		if r.listener != nil {
			_ = r.listener.Close()
		}
		r.mu.Lock()
		pairs := append([]*connectionPair(nil), r.pairs...)
		r.mu.Unlock()
		for _, pair := range pairs {
			_ = pair.downstream.Close()
			_ = pair.upstream.Close()
		}
		r.wg.Wait()
	})
	return nil
}

func (r *HalfOpen) BindAddr() string { return r.bindAddr }
func (r *HalfOpen) BindPort() int    { return r.bindPort }

func (p *connectionPair) isBlackholed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.blackholed
}
