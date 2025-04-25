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

package vnet

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/fatedier/golib/pool"
	"github.com/songgao/water/waterutil"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/xlog"
)

const (
	maxPacketSize = 1420
)

type Controller struct {
	addr string

	tun          io.ReadWriteCloser
	clientRouter *clientRouter // Route based on destination IP (client mode)
	serverRouter *serverRouter // Route based on source IP (server mode)
}

func NewController(cfg v1.VirtualNetConfig) *Controller {
	return &Controller{
		addr:         cfg.Address,
		clientRouter: newClientRouter(),
		serverRouter: newServerRouter(),
	}
}

func (c *Controller) Init() error {
	tunDevice, err := OpenTun(context.Background(), c.addr)
	if err != nil {
		return err
	}
	c.tun = tunDevice
	return nil
}

func (c *Controller) Run() error {
	conn := c.tun

	for {
		buf := pool.GetBuf(maxPacketSize)
		n, err := conn.Read(buf)
		if err != nil {
			pool.PutBuf(buf)
			log.Warnf("vnet read from tun error: %v", err)
			return err
		}

		c.handlePacket(buf[:n])
		pool.PutBuf(buf)
	}
}

// handlePacket processes a single packet. The caller is responsible for managing the buffer.
func (c *Controller) handlePacket(buf []byte) {
	log.Tracef("vnet read from tun [%d]: %s", len(buf), base64.StdEncoding.EncodeToString(buf))

	var src, dst net.IP
	switch {
	case waterutil.IsIPv4(buf):
		header, err := ipv4.ParseHeader(buf)
		if err != nil {
			log.Warnf("parse ipv4 header error: %v", err)
			return
		}
		src = header.Src
		dst = header.Dst
		log.Tracef("%s >> %s %d/%-4d %-4x %d",
			header.Src, header.Dst,
			header.Len, header.TotalLen, header.ID, header.Flags)
	case waterutil.IsIPv6(buf):
		header, err := ipv6.ParseHeader(buf)
		if err != nil {
			log.Warnf("parse ipv6 header error: %v", err)
			return
		}
		src = header.Src
		dst = header.Dst
		log.Tracef("%s >> %s %d %d",
			header.Src, header.Dst,
			header.PayloadLen, header.TrafficClass)
	default:
		log.Tracef("unknown packet, discarded(%d)", len(buf))
		return
	}

	targetConn, err := c.clientRouter.findConn(dst)
	if err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("write to client target conn error: %v", err)
		}
		return
	}

	targetConn, err = c.serverRouter.findConnBySrc(dst)
	if err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("write to server target conn error: %v", err)
		}
		return
	}

	log.Tracef("no route found for packet from %s to %s", src, dst)
}

func (c *Controller) Stop() error {
	return c.tun.Close()
}

// Client connection read loop
func (c *Controller) readLoopClient(ctx context.Context, conn io.ReadWriteCloser) {
	xl := xlog.FromContextSafe(ctx)
	defer func() {
		// Remove the route when read loop ends (connection closed)
		c.clientRouter.removeConnRoute(conn)
		conn.Close()
	}()

	for {
		data, err := ReadMessage(conn)
		if err != nil {
			xl.Warnf("client read error: %v", err)
			return
		}

		if len(data) == 0 {
			continue
		}

		switch {
		case waterutil.IsIPv4(data):
			header, err := ipv4.ParseHeader(data)
			if err != nil {
				xl.Warnf("parse ipv4 header error: %v", err)
				continue
			}
			xl.Tracef("%s >> %s %d/%-4d %-4x %d",
				header.Src, header.Dst,
				header.Len, header.TotalLen, header.ID, header.Flags)
		case waterutil.IsIPv6(data):
			header, err := ipv6.ParseHeader(data)
			if err != nil {
				xl.Warnf("parse ipv6 header error: %v", err)
				continue
			}
			xl.Tracef("%s >> %s %d %d",
				header.Src, header.Dst,
				header.PayloadLen, header.TrafficClass)
		default:
			xl.Tracef("unknown packet, discarded(%d)", len(data))
			continue
		}

		xl.Tracef("vnet write to tun (client) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		_, err = c.tun.Write(data)
		if err != nil {
			xl.Warnf("client write tun error: %v", err)
		}
	}
}

// Server connection read loop
func (c *Controller) readLoopServer(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	xl := xlog.FromContextSafe(ctx)
	defer func() {
		// Clean up all IP mappings associated with this connection when it closes
		c.serverRouter.cleanupConnIPs(conn)
		// Call the provided callback upon closure
		if onClose != nil {
			onClose()
		}
		conn.Close()
	}()

	for {
		data, err := ReadMessage(conn)
		if err != nil {
			xl.Warnf("server read error: %v", err)
			return
		}

		if len(data) == 0 {
			continue
		}

		// Register source IP to connection mapping
		if waterutil.IsIPv4(data) || waterutil.IsIPv6(data) {
			var src net.IP
			if waterutil.IsIPv4(data) {
				header, err := ipv4.ParseHeader(data)
				if err == nil {
					src = header.Src
					c.serverRouter.registerSrcIP(src, conn)
				}
			} else {
				header, err := ipv6.ParseHeader(data)
				if err == nil {
					src = header.Src
					c.serverRouter.registerSrcIP(src, conn)
				}
			}
		}

		xl.Tracef("vnet write to tun (server) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		_, err = c.tun.Write(data)
		if err != nil {
			xl.Warnf("server write tun error: %v", err)
		}
	}
}

// RegisterClientRoute registers a client route (based on destination IP CIDR)
// and starts the read loop
func (c *Controller) RegisterClientRoute(ctx context.Context, name string, routes []net.IPNet, conn io.ReadWriteCloser) {
	c.clientRouter.addRoute(name, routes, conn)
	go c.readLoopClient(ctx, conn)
}

// UnregisterClientRoute Remove client route from routing table
func (c *Controller) UnregisterClientRoute(name string) {
	c.clientRouter.delRoute(name)
}

// StartServerConnReadLoop starts the read loop for a server connection
// (dynamically associates with source IPs)
func (c *Controller) StartServerConnReadLoop(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	go c.readLoopServer(ctx, conn, onClose)
}

// ParseRoutes Convert route strings to IPNet objects
func ParseRoutes(routeStrings []string) ([]net.IPNet, error) {
	routes := make([]net.IPNet, 0, len(routeStrings))
	for _, r := range routeStrings {
		_, ipNet, err := net.ParseCIDR(r)
		if err != nil {
			return nil, fmt.Errorf("parse route %s error: %v", r, err)
		}
		routes = append(routes, *ipNet)
	}
	return routes, nil
}

// Client router (based on destination IP routing)
type clientRouter struct {
	routes map[string]*routeElement
	mu     sync.RWMutex
}

func newClientRouter() *clientRouter {
	return &clientRouter{
		routes: make(map[string]*routeElement),
	}
}

func (r *clientRouter) addRoute(name string, routes []net.IPNet, conn io.ReadWriteCloser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[name] = &routeElement{
		name:   name,
		routes: routes,
		conn:   conn,
	}
}

func (r *clientRouter) findConn(dst net.IP) (io.Writer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, re := range r.routes {
		for _, route := range re.routes {
			if route.Contains(dst) {
				return re.conn, nil
			}
		}
	}
	return nil, fmt.Errorf("no route found for destination %s", dst)
}

func (r *clientRouter) delRoute(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, name)
}

func (r *clientRouter) removeConnRoute(conn io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, re := range r.routes {
		if re.conn == conn {
			delete(r.routes, name)
			return
		}
	}
}

// Server router (based solely on source IP routing)
type serverRouter struct {
	srcIPConns map[string]io.Writer // Source IP string to connection mapping
	mu         sync.RWMutex
}

func newServerRouter() *serverRouter {
	return &serverRouter{
		srcIPConns: make(map[string]io.Writer),
	}
}

func (r *serverRouter) findConnBySrc(src net.IP) (io.Writer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, exists := r.srcIPConns[src.String()]
	if !exists {
		return nil, fmt.Errorf("no route found for source %s", src)
	}
	return conn, nil
}

func (r *serverRouter) registerSrcIP(src net.IP, conn io.Writer) {
	key := src.String()

	r.mu.RLock()
	existingConn, ok := r.srcIPConns[key]
	r.mu.RUnlock()

	// If the entry exists and the connection is the same, no need to do anything.
	if ok && existingConn == conn {
		return
	}

	// Acquire write lock to update the map.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring the write lock to handle potential race conditions.
	existingConn, ok = r.srcIPConns[key]
	if ok && existingConn == conn {
		return
	}

	r.srcIPConns[key] = conn
}

// cleanupConnIPs removes all IP mappings associated with the specified connection
func (r *serverRouter) cleanupConnIPs(conn io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find and delete all IP mappings pointing to this connection
	for ip, mappedConn := range r.srcIPConns {
		if mappedConn == conn {
			delete(r.srcIPConns, ip)
		}
	}
}

type routeElement struct {
	name   string
	routes []net.IPNet
	conn   io.ReadWriteCloser
}
