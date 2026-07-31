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

package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/proto/wire"
)

type sudpPathBenchmarkCase struct {
	name   string
	packet *msg.UDPPacket
}

var sudpPathBenchmarkBytesSink []byte

func sudpPathBenchmarkCases(payloadSize int) []sudpPathBenchmarkCase {
	content := bytes.Repeat([]byte{0x5a}, payloadSize)
	return []sudpPathBenchmarkCase{
		{
			name: "ipv4-remote",
			packet: &msg.UDPPacket{
				Content: content,
				RemoteAddr: &net.UDPAddr{
					IP: net.ParseIP("192.0.2.1"), Port: 12345,
				},
			},
		},
		{
			name: "ipv4-local-remote",
			packet: &msg.UDPPacket{
				Content: content,
				LocalAddr: &net.UDPAddr{
					IP: net.ParseIP("192.0.2.2"), Port: 23456,
				},
				RemoteAddr: &net.UDPAddr{
					IP: net.ParseIP("192.0.2.1"), Port: 12345,
				},
			},
		},
		{
			name: "ipv6-remote",
			packet: &msg.UDPPacket{
				Content: content,
				RemoteAddr: &net.UDPAddr{
					IP: net.ParseIP("2001:db8::1"), Port: 12345,
				},
			},
		},
		{
			name: "ipv6-local-remote",
			packet: &msg.UDPPacket{
				Content: content,
				LocalAddr: &net.UDPAddr{
					IP: net.ParseIP("2001:db8::2"), Port: 23456, Zone: "bench0",
				},
				RemoteAddr: &net.UDPAddr{
					IP: net.ParseIP("2001:db8::1"), Port: 12345, Zone: "bench1",
				},
			},
		},
	}
}

type sudpPathReadWriter struct {
	reader bytes.Reader
}

func (rw *sudpPathReadWriter) Read(p []byte) (int, error)  { return rw.reader.Read(p) }
func (rw *sudpPathReadWriter) Write(p []byte) (int, error) { return len(p), nil }
func (rw *sudpPathReadWriter) Reset(p []byte)              { rw.reader.Reset(p) }

func sudpPathWireBytes(b testing.TB, packet *msg.UDPPacket, codec string) []byte {
	b.Helper()
	var buf bytes.Buffer
	rw, err := msg.NewUDPPacketReadWriter(&buf, wire.ProtocolV2, codec)
	if err != nil {
		b.Fatal(err)
	}
	if err := rw.WriteMsg(packet); err != nil {
		b.Fatal(err)
	}
	return append([]byte(nil), buf.Bytes()...)
}

func sudpPathCopyFrame(dst, src []byte) []byte {
	copy(dst, src)
	return dst
}

func BenchmarkSUDPInMemoryFrameCopy(b *testing.B) {
	// This is an in-memory copy of an already encoded frame. It is a proxy for
	// frame-size-dependent copy work, not a benchmark of libio.Join or sockets.
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range sudpPathBenchmarkCases(payloadSize) {
			for _, codec := range []struct {
				name  string
				value string
			}{
				{name: "json", value: ""},
				{name: "binary", value: wire.UDPPacketCodecBinary},
			} {
				b.Run(fmt.Sprintf("payload-%d/%s/%s", payloadSize, tc.name, codec.name), func(b *testing.B) {
					encoded := sudpPathWireBytes(b, tc.packet, codec.value)
					dst := make([]byte, len(encoded))
					b.SetBytes(int64(len(encoded)))
					for b.Loop() {
						dst = sudpPathCopyFrame(dst, encoded)
					}
					if !bytes.Equal(dst, encoded) {
						b.Fatal("copied frame does not match source")
					}
					sudpPathBenchmarkBytesSink = dst
				})
			}
		}
	}
}

func BenchmarkSUDPEndpointCodecPair(b *testing.B) {
	// This measures an in-memory decode and re-encode with the same codec. It
	// does not include the live SUDP server path, sockets, goroutines, or I/O.
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range sudpPathBenchmarkCases(payloadSize) {
			for _, codec := range []struct {
				name  string
				value string
			}{
				{name: "json", value: ""},
				{name: "binary", value: wire.UDPPacketCodecBinary},
			} {
				b.Run(fmt.Sprintf("payload-%d/%s/%s", payloadSize, tc.name, codec.name), func(b *testing.B) {
					encoded := sudpPathWireBytes(b, tc.packet, codec.value)
					from := &sudpPathReadWriter{}
					to := &bytes.Buffer{}
					fromRW, err := msg.NewUDPPacketReadWriter(from, wire.ProtocolV2, codec.value)
					if err != nil {
						b.Fatal(err)
					}
					toRW, err := msg.NewUDPPacketReadWriter(to, wire.ProtocolV2, codec.value)
					if err != nil {
						b.Fatal(err)
					}
					b.SetBytes(int64(len(encoded)))
					for b.Loop() {
						from.Reset(encoded)
						to.Reset()
						m, err := fromRW.ReadMsg()
						if err != nil {
							b.Fatal(err)
						}
						if err := toRW.WriteMsg(m); err != nil {
							b.Fatal(err)
						}
					}
					if !bytes.Equal(to.Bytes(), encoded) {
						b.Fatalf("re-encoded packet mismatch: got %d bytes, want %d", to.Len(), len(encoded))
					}
					sudpPathBenchmarkBytesSink = to.Bytes()
				})
			}
		}
	}
}

func BenchmarkSUDPMixedCodecTranscodeModel(b *testing.B) {
	// This exercises the real codec decode/re-encode pair used by the mixed
	// bridge, excluding sockets, goroutines, crypto, compression, and framing I/O.
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range sudpPathBenchmarkCases(payloadSize) {
			for _, direction := range []struct {
				name string
				from string
				to   string
			}{
				{name: "json-to-binary", from: "", to: wire.UDPPacketCodecBinary},
				{name: "binary-to-json", from: wire.UDPPacketCodecBinary, to: ""},
			} {
				b.Run(fmt.Sprintf("payload-%d/%s/%s", payloadSize, tc.name, direction.name), func(b *testing.B) {
					encoded := sudpPathWireBytes(b, tc.packet, direction.from)
					expected := sudpPathWireBytes(b, tc.packet, direction.to)
					from := &sudpPathReadWriter{}
					to := &bytes.Buffer{}
					fromRW, err := msg.NewUDPPacketReadWriter(from, wire.ProtocolV2, direction.from)
					if err != nil {
						b.Fatal(err)
					}
					toRW, err := msg.NewUDPPacketReadWriter(to, wire.ProtocolV2, direction.to)
					if err != nil {
						b.Fatal(err)
					}
					b.SetBytes(int64(len(encoded)))
					for b.Loop() {
						from.Reset(encoded)
						to.Reset()
						m, err := fromRW.ReadMsg()
						if err != nil {
							b.Fatal(err)
						}
						if err := toRW.WriteMsg(m); err != nil {
							b.Fatal(err)
						}
					}
					if !bytes.Equal(to.Bytes(), expected) {
						b.Fatalf("transcoded packet mismatch: got %d bytes, want %d", to.Len(), len(expected))
					}
					sudpPathBenchmarkBytesSink = to.Bytes()
				})
			}
		}
	}
}

var _ io.ReadWriter = (*sudpPathReadWriter)(nil)
