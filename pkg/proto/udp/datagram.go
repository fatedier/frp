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

package udp

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	quic "github.com/quic-go/quic-go"
)

// QUIC datagram relay (RFC 9221) for udp proxies. Every datagram carries a
// compact binary header so multiple proxies can share one quic connection:
//
//	[0]      version (dgramFrameVersion)
//	[1]      len(proxyName)
//	[2:2+n]  proxyName
//	[+0]     len(addr)
//	[+1:+1+a] addr — the udp user address "ip:port" (direction-independent:
//	          frps->frpc it identifies the sender, frpc->frps the reply target)
//	[rest]   raw udp payload
//
// Unlike msg.UDPPacket over the work connection, payloads are neither
// JSON-wrapped nor base64-encoded, and delivery is unreliable and unordered:
// a lost datagram is simply lost, which is exactly what the tunneled
// protocols (DTLS, WireGuard, RTP) expect from a udp path.
const (
	dgramFrameVersion = 0xF1

	// dgramFrameBudget caps the whole encoded frame. quic-go's SendDatagram
	// size check uses the current MTU estimate WITHOUT accounting for
	// packet headers/AEAD overhead: frames in that gray zone are accepted
	// and then silently dropped at packing time — a deterministic
	// blackhole for those sizes. Stay well below the minimum QUIC packet
	// budget (1252-byte initial packets minus ~50 bytes of overhead) so a
	// frame that passes here always fits; anything larger falls back to
	// the reliable stream.
	dgramFrameBudget = 1180
)

// ErrDatagramTooLarge is returned by Send when the frame exceeds what the
// QUIC connection can currently carry; callers should fall back to the
// reliable work-connection stream for that packet.
var ErrDatagramTooLarge = errors.New("udp datagram too large for quic path")

type dgramHandler func(remoteAddr *net.UDPAddr, payload []byte)

// DatagramMux fans incoming QUIC datagrams of one connection out to the
// registered proxies, and encodes outgoing ones.
type DatagramMux struct {
	qc *quic.Conn

	mu       sync.RWMutex
	handlers map[string]dgramHandler
}

var (
	dgramMusMu sync.Mutex
	dgramMuxes = map[*quic.Conn]*DatagramMux{}
)

// DatagramMuxFor returns the mux of a quic connection, creating it and
// starting its receive loop on first use. The mux removes itself when the
// connection dies.
func DatagramMuxFor(qc *quic.Conn) *DatagramMux {
	dgramMusMu.Lock()
	defer dgramMusMu.Unlock()
	if m, ok := dgramMuxes[qc]; ok {
		return m
	}
	m := &DatagramMux{
		qc:       qc,
		handlers: make(map[string]dgramHandler),
	}
	dgramMuxes[qc] = m
	go m.receiveLoop()
	return m
}

func (m *DatagramMux) receiveLoop() {
	defer func() {
		dgramMusMu.Lock()
		delete(dgramMuxes, m.qc)
		dgramMusMu.Unlock()
	}()
	ctx := m.qc.Context()
	for {
		data, err := m.qc.ReceiveDatagram(ctx)
		if err != nil {
			return
		}
		name, addr, payload, err := decodeDgramFrame(data)
		if err != nil {
			continue
		}
		m.mu.RLock()
		h := m.handlers[name]
		m.mu.RUnlock()
		if h != nil {
			h(addr, payload)
		}
	}
}

// Register installs the receive handler for a proxy, replacing any previous
// one (a new work connection re-registers with its fresh channels).
func (m *DatagramMux) Register(name string, h dgramHandler) {
	m.mu.Lock()
	m.handlers[name] = h
	m.mu.Unlock()
}

func (m *DatagramMux) Unregister(name string) {
	m.mu.Lock()
	delete(m.handlers, name)
	m.mu.Unlock()
}

// Send relays one udp packet as an unreliable QUIC datagram. Returns
// ErrDatagramTooLarge when it doesn't fit the connection's current datagram
// budget (caller falls back to the stream).
func (m *DatagramMux) Send(name string, remoteAddr *net.UDPAddr, payload []byte) error {
	frame, err := encodeDgramFrame(name, remoteAddr, payload)
	if err != nil {
		return err
	}
	err = m.qc.SendDatagram(frame)
	if err != nil {
		var tooLarge *quic.DatagramTooLargeError
		if errors.As(err, &tooLarge) {
			return ErrDatagramTooLarge
		}
		return err
	}
	return nil
}

func encodeDgramFrame(name string, remoteAddr *net.UDPAddr, payload []byte) ([]byte, error) {
	if remoteAddr == nil {
		return nil, errors.New("udp datagram without remote address")
	}
	addr := remoteAddr.String()
	if len(name) > 255 || len(addr) > 255 {
		return nil, fmt.Errorf("udp datagram header field too long: name %d, addr %d", len(name), len(addr))
	}
	// Enforce the budget before building the frame: MTU-sized payloads
	// commonly exceed it and fall back to the stream, so don't pay an
	// allocation and copy just to throw the frame away.
	frameLen := 3 + len(name) + len(addr) + len(payload)
	if frameLen > dgramFrameBudget {
		return nil, ErrDatagramTooLarge
	}
	frame := make([]byte, 0, frameLen)
	frame = append(frame, dgramFrameVersion, byte(len(name)))
	frame = append(frame, name...)
	frame = append(frame, byte(len(addr)))
	frame = append(frame, addr...)
	frame = append(frame, payload...)
	return frame, nil
}

func decodeDgramFrame(data []byte) (name string, remoteAddr *net.UDPAddr, payload []byte, err error) {
	if len(data) < 3 || data[0] != dgramFrameVersion {
		err = errors.New("malformed udp datagram frame")
		return
	}
	nameLen := int(data[1])
	if len(data) < 2+nameLen+1 {
		err = errors.New("malformed udp datagram frame: name")
		return
	}
	name = string(data[2 : 2+nameLen])
	addrLen := int(data[2+nameLen])
	addrStart := 2 + nameLen + 1
	if len(data) < addrStart+addrLen {
		err = errors.New("malformed udp datagram frame: addr")
		return
	}
	// The address comes off the wire; accept only numeric "ip:port".
	// net.ResolveUDPAddr would resolve hostnames, letting a peer stall the
	// connection's shared receive loop on DNS lookups.
	ap, perr := netip.ParseAddrPort(string(data[addrStart : addrStart+addrLen]))
	if perr != nil {
		err = errors.New("malformed udp datagram frame: addr not numeric ip:port")
		return
	}
	remoteAddr = net.UDPAddrFromAddrPort(ap)
	payload = data[addrStart+addrLen:]
	return
}
