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

const maxPacketSize = 1420

// Controller manages the virtual network TUN interface and routes packets
type Controller struct {
	addr string

	tun          io.ReadWriteCloser
	clientRouter *clientRouter // routes packets by destination IP (client mode)
	serverRouter *serverRouter // routes packets by source IP (server mode)
}

// NewController creates a new Controller based on the provided configuration.
func NewController(cfg v1.VirtualNetConfig) *Controller {
	return &Controller{
		addr:         cfg.Address,
		clientRouter: newClientRouter(),
		serverRouter: newServerRouter(),
	}
}

// Init opens the TUN device with the configured address.
func (c *Controller) Init() error {
	tunDevice, err := OpenTun(context.Background(), c.addr)
	if err != nil {
		return err
	}
	c.tun = tunDevice
	return nil
}

// Run continuously reads packets from the TUN device and processes them.
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

// handlePacket parses the packet header and forwards it to the appropriate route.
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
		src, dst = header.Src, header.Dst
		log.Tracef("%s >> %s %d/%-4d %-4x %d", src, dst, header.Len, header.TotalLen, header.ID, header.Flags)

	case waterutil.IsIPv6(buf):
		header, err := ipv6.ParseHeader(buf)
		if err != nil {
			log.Warnf("parse ipv6 header error: %v", err)
			return
		}
		src, dst = header.Src, header.Dst
		log.Tracef("%s >> %s %d %d", src, dst, header.PayloadLen, header.TrafficClass)

	default:
		log.Tracef("unknown packet, discarded (%d bytes)", len(buf))
		return
	}

	// Try client route (based on destination IP)
	if targetConn, err := c.clientRouter.findConn(dst); err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("write to client target conn error: %v", err)
		}
		return
	}

	// Try server route (based on source IP)
	if targetConn, err := c.serverRouter.findConnBySrc(dst); err == nil {
		if err := WriteMessage(targetConn, buf); err != nil {
			log.Warnf("write to server target conn error: %v", err)
		}
		return
	}

	log.Tracef("no route found for packet from %s to %s", src, dst)
}

// Stop closes the TUN interface.
func (c *Controller) Stop() error {
	return c.tun.Close()
}

// readLoopClient reads packets from a client connection and writes them to the TUN device.
func (c *Controller) readLoopClient(ctx context.Context, conn io.ReadWriteCloser) {
	xl := xlog.FromContextSafe(ctx)
	defer func() {
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

		logPacketHeader(xl, data)

		xl.Tracef("vnet write to tun (client) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		if _, err := c.tun.Write(data); err != nil {
			xl.Warnf("client write tun error: %v", err)
		}
	}
}

// readLoopServer reads packets from a server connection and writes them to the TUN device,
// while maintaining source IP to connection mappings.
func (c *Controller) readLoopServer(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	xl := xlog.FromContextSafe(ctx)
	defer func() {
		c.serverRouter.cleanupConnIPs(conn)
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

		registerSourceIP(c.serverRouter, data)

		xl.Tracef("vnet write to tun (server) [%d]: %s", len(data), base64.StdEncoding.EncodeToString(data))
		if _, err := c.tun.Write(data); err != nil {
			xl.Warnf("server write tun error: %v", err)
		}
	}
}

// RegisterClientRoute adds a client route and starts the read loop for the connection.
func (c *Controller) RegisterClientRoute(ctx context.Context, name string, routes []net.IPNet, conn io.ReadWriteCloser) {
	c.clientRouter.addRoute(name, routes, conn)
	go c.readLoopClient(ctx, conn)
}

// UnregisterClientRoute removes a client route.
func (c *Controller) UnregisterClientRoute(name string) {
	c.clientRouter.delRoute(name)
}

// StartServerConnReadLoop starts the read loop for a server connection with cleanup on close.
func (c *Controller) StartServerConnReadLoop(ctx context.Context, conn io.ReadWriteCloser, onClose func()) {
	go c.readLoopServer(ctx, conn, onClose)
}

// ParseRoutes parses CIDR route strings into net.IPNet objects.
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

// Helper to log IP packet header information
func logPacketHeader(xl xlog.Logger, data []byte) {
	switch {
	case waterutil.IsIPv4(data):
		if header, err := ipv4.ParseHeader(data); err == nil {
			xl.Tracef("%s >> %s %d/%-4d %-4x %d", header.Src, header.Dst, header.Len, header.TotalLen, header.ID, header.Flags)
		}
	case waterutil.IsIPv6(data):
		if header, err := ipv6.ParseHeader(data); err == nil {
			xl.Tracef("%s >> %s %d %d", header.Src, header.Dst, header.PayloadLen, header.TrafficClass)
		}
	default:
		xl.Tracef("unknown packet, discarded(%d)", len(data))
	}
}

// Helper to register source IP for server router
func registerSourceIP(r *serverRouter, data []byte) {
	if waterutil.IsIPv4(data) {
		if header, err := ipv4.ParseHeader(data); err == nil {
			r.registerSrcIP(header.Src, nil) // nil or actual connection if available
		}
	} else if waterutil.IsIPv6(data) {
		if header, err := ipv6.ParseHeader(data); err == nil {
			r.registerSrcIP(header.Src, nil)
		}
	}
}

// ----------- clientRouter ------------

// clientRouter routes packets based on destination IP.
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
	r.routes[name] = &routeElement{name: name, routes: routes, conn: conn}
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

// ----------- serverRouter ------------

// serverRouter routes packets based on source IP.
type serverRouter struct {
	srcIPConns map[string]io.Writer
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

	if ok && existingConn == conn {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after locking to avoid race condition
	existingConn, ok = r.srcIPConns[key]
	if ok && existingConn == conn {
		return
	}

	r.srcIPConns[key] = conn
}

func (r *serverRouter) cleanupConnIPs(conn io.Writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for ip, mappedConn := range r.srcIPConns {
		if mappedConn == conn {
			delete(r.srcIPConns, ip)
		}
	}
}

// ----------- routeElement ------------

// routeElement associates a route name with IP networks and a connection.
type routeElement struct {
	name   string
	routes []net.IPNet
	conn   io.ReadWriteCloser
}
