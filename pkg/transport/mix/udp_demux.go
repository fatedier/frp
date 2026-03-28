package mix

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	udpPeerRouteTTL           = 5 * time.Minute
	udpPeerRoutePruneInterval = 1 * time.Minute
)

type packet struct {
	data []byte
	addr net.Addr
}

type demuxPacketConn struct {
	parent *UDPDemux
	name   string
	ch     chan packet

	mu           sync.RWMutex
	readDeadline time.Time
	closed       bool
}

func (c *demuxPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		c.mu.RLock()
		deadline := c.readDeadline
		closed := c.closed
		c.mu.RUnlock()
		if closed {
			return 0, nil, net.ErrClosed
		}

		var timer <-chan time.Time
		if !deadline.IsZero() {
			d := time.Until(deadline)
			if d <= 0 {
				return 0, nil, net.ErrClosed
			}
			timer = time.After(d)
		}

		select {
		case pkt, ok := <-c.ch:
			if !ok {
				return 0, nil, net.ErrClosed
			}
			n := copy(p, pkt.data)
			return n, pkt.addr, nil
		case <-timer:
			return 0, nil, net.ErrClosed
		}
	}
}

func (c *demuxPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return c.parent.conn.WriteTo(p, addr)
}

func (c *demuxPacketConn) enqueue(pkt packet) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.ch <- pkt:
		return true
	default:
		return false
	}
}

func (c *demuxPacketConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.ch)
	return nil
}

func (c *demuxPacketConn) LocalAddr() net.Addr {
	return c.parent.conn.LocalAddr()
}

func (c *demuxPacketConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	return nil
}

func (c *demuxPacketConn) SetReadDeadline(t time.Time) error {
	return c.SetDeadline(t)
}

func (c *demuxPacketConn) SetWriteDeadline(time.Time) error {
	return nil
}

type UDPDemux struct {
	conn net.PacketConn

	mu                sync.RWMutex
	routeFn           func([]byte) string
	peers             map[string]peerRoute
	conns             map[string]*demuxPacketConn
	lastPeerPruneTime time.Time
	closed            bool
}

type peerRoute struct {
	child    *demuxPacketConn
	lastSeen time.Time
}

func NewUDPDemux(conn net.PacketConn, routeFn func([]byte) string, names ...string) *UDPDemux {
	d := &UDPDemux{
		conn:              conn,
		routeFn:           routeFn,
		peers:             make(map[string]peerRoute),
		conns:             make(map[string]*demuxPacketConn, len(names)),
		lastPeerPruneTime: time.Now(),
	}
	for _, name := range names {
		d.conns[name] = &demuxPacketConn{
			parent: d,
			name:   name,
			ch:     make(chan packet, 256),
		}
	}
	return d
}

func (d *UDPDemux) Conn(name string) net.PacketConn {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.conns[name]
}

func (d *UDPDemux) Serve() error {
	buf := make([]byte, 64*1024)
	for {
		n, addr, err := d.conn.ReadFrom(buf)
		if err != nil {
			return err
		}
		key := addr.String()
		now := time.Now()

		d.mu.Lock()
		if now.Sub(d.lastPeerPruneTime) >= udpPeerRoutePruneInterval {
			d.pruneStalePeersLocked(now)
			d.lastPeerPruneTime = now
		}

		route := d.peers[key]
		child := route.child
		if child == nil {
			name := d.routeFn(buf[:n])
			child = d.conns[name]
			if child == nil {
				d.mu.Unlock()
				continue
			}
			route.child = child
		}
		route.lastSeen = now
		d.peers[key] = route
		d.mu.Unlock()

		pkt := packet{
			data: append([]byte(nil), buf[:n]...),
			addr: addr,
		}
		_ = child.enqueue(pkt)
	}
}

func (d *UDPDemux) pruneStalePeersLocked(now time.Time) {
	for key, route := range d.peers {
		if now.Sub(route.lastSeen) > udpPeerRouteTTL {
			delete(d.peers, key)
		}
	}
}

func (d *UDPDemux) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	for _, child := range d.conns {
		_ = child.Close()
	}
	return d.conn.Close()
}

func RouteQUICThenKCP(packet []byte) string {
	if len(packet) == 0 {
		return ""
	}
	// QUIC Initial packets always carry both Header Form (0x80) and Fixed Bit (0x40).
	if packet[0]&0x80 != 0 && packet[0]&0x40 != 0 {
		return "quic"
	}
	return "kcp"
}

func MustPacketConn(d *UDPDemux, name string) net.PacketConn {
	conn := d.Conn(name)
	if conn == nil {
		panic(fmt.Sprintf("missing demux packet conn %q", name))
	}
	return conn
}
