// Copyright 2024 The frp Authors
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

package group

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/util/vhost"
	"github.com/fatedier/frp/server/ports"
)

// MinecraftGroupController lazily opens a shared Minecraft host-routing muxer per
// public port that clients declare (the "mc" proxy type's remotePort), so frps
// needs no static port configuration at all. Multiple proxies — even from
// different clients — that declare the same remotePort share a single listener
// and are routed by the hostname in the Minecraft handshake. The port is
// reserved through the TCP port manager (so allowPorts and conflicts with other
// proxies are enforced) and released once the last proxy on it goes away.
type MinecraftGroupController struct {
	mu            sync.Mutex
	muxers        map[int]*minecraftPortMux
	proxyBindAddr string
	portManager   *ports.Manager
	timeout       time.Duration
}

type minecraftPortMux struct {
	muxer *vhost.MinecraftMuxer
	refs  int
}

func NewMinecraftGroupController(proxyBindAddr string, portManager *ports.Manager, timeout time.Duration) *MinecraftGroupController {
	return &MinecraftGroupController{
		muxers:        make(map[int]*minecraftPortMux),
		proxyBindAddr: proxyBindAddr,
		portManager:   portManager,
		timeout:       timeout,
	}
}

// Listen registers routeConfig.Domain on the shared muxer for `port`, creating
// the muxer (and reserving the port) on first use. The returned listener's Close
// removes the domain and, when the last domain on that port is gone, tears down
// the muxer and releases the port.
func (ctl *MinecraftGroupController) Listen(ctx context.Context, port int, routeConfig vhost.RouteConfig) (net.Listener, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	entry, ok := ctl.muxers[port]
	if !ok {
		realPort, err := ctl.portManager.Acquire(minecraftPortName(port), port)
		if err != nil {
			return nil, fmt.Errorf("acquire minecraft port [%d] error: %v", port, err)
		}
		address := net.JoinHostPort(ctl.proxyBindAddr, strconv.Itoa(realPort))
		l, err := net.Listen("tcp", address)
		if err != nil {
			ctl.portManager.Release(realPort)
			return nil, err
		}
		muxer, err := vhost.NewMinecraftMuxer(l, ctl.timeout)
		if err != nil {
			_ = l.Close()
			ctl.portManager.Release(realPort)
			return nil, err
		}
		entry = &minecraftPortMux{muxer: muxer}
		ctl.muxers[port] = entry
	}

	vl, err := entry.muxer.Listen(ctx, &routeConfig)
	if err != nil {
		// Roll back a freshly created, still-unreferenced port muxer.
		if entry.refs == 0 {
			entry.muxer.Close()
			ctl.portManager.Release(port)
			delete(ctl.muxers, port)
		}
		return nil, err
	}
	entry.refs++
	return &minecraftGroupListener{Listener: vl, ctl: ctl, port: port}, nil
}

// CloseAll tears down every remaining port muxer. Used on server shutdown as a
// safety net; in normal operation proxies release their ports as they close.
func (ctl *MinecraftGroupController) CloseAll() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	for port, entry := range ctl.muxers {
		entry.muxer.Close()
		ctl.portManager.Release(port)
		delete(ctl.muxers, port)
	}
}

func (ctl *MinecraftGroupController) release(port int) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()
	entry, ok := ctl.muxers[port]
	if !ok {
		return
	}
	entry.refs--
	if entry.refs <= 0 {
		entry.muxer.Close()
		ctl.portManager.Release(port)
		delete(ctl.muxers, port)
	}
}

func minecraftPortName(port int) string {
	return fmt.Sprintf("@minecraft-%d", port)
}

// minecraftGroupListener wraps a vhost per-domain listener so closing it also
// decrements the shared port's reference count.
type minecraftGroupListener struct {
	net.Listener
	ctl  *MinecraftGroupController
	port int
	once sync.Once
}

func (l *minecraftGroupListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() { l.ctl.release(l.port) })
	return err
}
