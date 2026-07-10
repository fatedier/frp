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

package vhost

import (
	"errors"
	"io"
	"net"
	"strings"
	"time"

	libnet "github.com/fatedier/golib/net"
)

// maxMinecraftHandshakeSize bounds how many payload bytes we read while looking
// for the routing hostname, protecting against a client that declares a huge
// packet length. A real handshake is tiny; even Forge/BungeeCord suffixes stay
// well under this.
const maxMinecraftHandshakeSize = 32 * 1024

type MinecraftMuxer struct {
	*Muxer
}

// NewMinecraftMuxer builds a vhost muxer that routes incoming Minecraft (Java
// Edition) connections by the "server address" field of their first handshake
// packet — the hostname the player typed into the game client. This is the
// SNI-style equivalent for the Minecraft protocol, mirroring NewHTTPSMuxer.
func NewMinecraftMuxer(listener net.Listener, timeout time.Duration) (*MinecraftMuxer, error) {
	mux, err := NewMuxer(listener, GetMinecraftHostname, timeout)
	if err != nil {
		return nil, err
	}
	mux.SetFailHookFunc(minecraftFailed)
	return &MinecraftMuxer{mux}, nil
}

// GetMinecraftHostname peeks the Minecraft handshake at the start of the
// connection, extracts the routing hostname, and returns a SharedConn that
// replays every consumed byte so the backend server still receives the intact
// handshake.
func GetMinecraftHostname(c net.Conn) (net.Conn, map[string]string, error) {
	reqInfoMap := make(map[string]string, 1)
	sc, rd := libnet.NewSharedConn(c)

	host, err := readHandshakeServerHost(rd)
	if err != nil {
		return nil, reqInfoMap, err
	}
	reqInfoMap["Host"] = host
	return sc, reqInfoMap, nil
}

func minecraftFailed(c net.Conn) {
	_ = c.Close()
}

// readHandshakeServerHost reads exactly the first Minecraft packet (a VarInt
// length prefix followed by the payload) and returns the server address used
// for routing. Forge/BungeeCord append NUL-separated data to the address; only
// the part before the first NUL is the route key.
//
// Handshake payload layout: packetID(=0) | protocolVersion | serverAddress |
// serverPort(u16) | nextState. We only need up to serverAddress.
func readHandshakeServerHost(r io.Reader) (string, error) {
	length, err := readVarIntStream(r)
	if err != nil {
		return "", err
	}
	if length <= 0 || length > maxMinecraftHandshakeSize {
		return "", errors.New("invalid minecraft handshake length")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", err
	}

	packetID, n, err := readVarIntBuf(payload)
	if err != nil || packetID != 0 {
		return "", errors.New("not a minecraft handshake packet")
	}
	payload = payload[n:]
	// protocol version — read to advance, value unused for routing.
	if _, n, err = readVarIntBuf(payload); err != nil {
		return "", err
	}
	payload = payload[n:]
	host, _, err := readMinecraftString(payload)
	if err != nil {
		return "", err
	}
	if i := strings.IndexByte(host, 0); i != -1 {
		host = host[:i]
	}
	if host == "" {
		return "", errors.New("empty minecraft server address")
	}
	return host, nil
}

// readVarIntStream decodes a Minecraft VarInt from a streaming reader, reading
// one byte at a time (a VarInt is at most 5 bytes).
func readVarIntStream(r io.Reader) (int, error) {
	var value int
	var b [1]byte
	for i := 0; i < 5; i++ {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= int(b[0]&0x7f) << (7 * i)
		if b[0]&0x80 == 0 {
			return value, nil
		}
	}
	return 0, errors.New("varint too long")
}

// readVarIntBuf decodes a VarInt from an in-memory buffer: 7 bits per byte, high
// bit set means another byte follows.
func readVarIntBuf(buf []byte) (int, int, error) {
	var value int
	for i := 0; i < 5; i++ {
		if i >= len(buf) {
			return 0, 0, errors.New("incomplete varint")
		}
		b := buf[i]
		value |= int(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return value, i + 1, nil
		}
	}
	return 0, 0, errors.New("varint too long")
}

// readMinecraftString decodes a VarInt-length-prefixed UTF-8 string.
func readMinecraftString(buf []byte) (string, int, error) {
	length, n, err := readVarIntBuf(buf)
	if err != nil {
		return "", 0, err
	}
	if length < 0 || len(buf[n:]) < length {
		return "", 0, errors.New("incomplete string")
	}
	return string(buf[n : n+length]), n + length, nil
}
