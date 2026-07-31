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

package msg

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"testing"

	"github.com/fatedier/frp/pkg/proto/wire"
)

type udpBenchmarkCase struct {
	name   string
	packet *UDPPacket
}

var (
	udpBenchmarkBytesSink   []byte
	udpBenchmarkMessageSink Message
)

func udpBenchmarkCases(payloadSize int) []udpBenchmarkCase {
	content := bytes.Repeat([]byte{0x5a}, payloadSize)
	return []udpBenchmarkCase{
		{
			name: "ipv4-remote",
			packet: &UDPPacket{
				Content:    content,
				RemoteAddr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
			},
		},
		{
			name: "ipv4-local-remote",
			packet: &UDPPacket{
				Content:    content,
				LocalAddr:  &net.UDPAddr{IP: net.ParseIP("192.0.2.2"), Port: 23456},
				RemoteAddr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
			},
		},
		{
			name: "ipv6-remote",
			packet: &UDPPacket{
				Content:    content,
				RemoteAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 12345},
			},
		},
		{
			name: "ipv6-local-remote",
			packet: &UDPPacket{
				Content:    content,
				LocalAddr:  &net.UDPAddr{IP: net.ParseIP("2001:db8::2"), Port: 23456, Zone: "bench0"},
				RemoteAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 12345, Zone: "bench1"},
			},
		},
	}
}

func TestUDPPacketV2FrameSizes(t *testing.T) {
	t.Logf("environment go=%s goos=%s goarch=%s gomaxprocs=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range udpBenchmarkCases(payloadSize) {
			jsonFrame := udpBenchmarkWireBytes(t, tc.packet, "")
			binaryFrame := udpBenchmarkWireBytes(t, tc.packet, wire.UDPPacketCodecBinary)
			saving := 100 * float64(len(jsonFrame)-len(binaryFrame)) / float64(len(jsonFrame))
			t.Logf("frame payload=%d case=%s json_bytes=%d binary_bytes=%d binary_saving_pct=%.2f", payloadSize, tc.name, len(jsonFrame), len(binaryFrame), saving)
		}
	}
}

func udpBenchmarkWireBytes(t testing.TB, packet *UDPPacket, codec string) []byte {
	t.Helper()
	var buf bytes.Buffer
	rw, err := NewUDPPacketReadWriter(&buf, wire.ProtocolV2, codec)
	if err != nil {
		t.Fatalf("create UDP read writer: %v", err)
	}
	if err := rw.WriteMsg(packet); err != nil {
		t.Fatalf("write UDP packet: %v", err)
	}
	return append([]byte(nil), buf.Bytes()...)
}

type udpBenchmarkReadWriter struct {
	reader bytes.Reader
}

func (rw *udpBenchmarkReadWriter) Read(p []byte) (int, error) {
	return rw.reader.Read(p)
}

func (rw *udpBenchmarkReadWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (rw *udpBenchmarkReadWriter) Reset(p []byte) {
	rw.reader.Reset(p)
}

func udpBenchmarkValidatePacket(b testing.TB, got, want *UDPPacket) {
	b.Helper()
	if !bytes.Equal(got.Content, want.Content) || !udpBenchmarkUDPAddrEqual(got.LocalAddr, want.LocalAddr) ||
		!udpBenchmarkUDPAddrEqual(got.RemoteAddr, want.RemoteAddr) {
		b.Fatalf("decoded packet mismatch: got %+v, want %+v", got, want)
	}
}

func udpBenchmarkUDPAddrEqual(got, want *net.UDPAddr) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.IP.Equal(want.IP) && got.Port == want.Port && got.Zone == want.Zone
}

func BenchmarkUDPPacketV2CodecWrite(b *testing.B) {
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range udpBenchmarkCases(payloadSize) {
			for _, codec := range []struct {
				name  string
				value string
			}{
				{name: "json", value: ""},
				{name: "binary", value: wire.UDPPacketCodecBinary},
			} {
				b.Run(fmt.Sprintf("payload-%d/%s/%s", payloadSize, tc.name, codec.name), func(b *testing.B) {
					var buf bytes.Buffer
					rw, err := NewUDPPacketReadWriter(&buf, wire.ProtocolV2, codec.value)
					if err != nil {
						b.Fatal(err)
					}
					expected := udpBenchmarkWireBytes(b, tc.packet, codec.value)
					b.SetBytes(int64(len(expected)))
					for b.Loop() {
						buf.Reset()
						if err := rw.WriteMsg(tc.packet); err != nil {
							b.Fatal(err)
						}
					}
					if !bytes.Equal(buf.Bytes(), expected) {
						b.Fatalf("encoded packet mismatch: got %d bytes, want %d", buf.Len(), len(expected))
					}
					udpBenchmarkBytesSink = buf.Bytes()
				})
			}
		}
	}
}

func BenchmarkUDPPacketV2CodecRead(b *testing.B) {
	for _, payloadSize := range []int{64, 512, 1200, 1472} {
		for _, tc := range udpBenchmarkCases(payloadSize) {
			for _, codec := range []struct {
				name  string
				value string
			}{
				{name: "json", value: ""},
				{name: "binary", value: wire.UDPPacketCodecBinary},
			} {
				b.Run(fmt.Sprintf("payload-%d/%s/%s", payloadSize, tc.name, codec.name), func(b *testing.B) {
					encoded := udpBenchmarkWireBytes(b, tc.packet, codec.value)
					stream := &udpBenchmarkReadWriter{}
					rw, err := NewUDPPacketReadWriter(stream, wire.ProtocolV2, codec.value)
					if err != nil {
						b.Fatal(err)
					}
					var decoded Message
					b.SetBytes(int64(len(encoded)))
					for b.Loop() {
						stream.Reset(encoded)
						decoded, err = rw.ReadMsg()
						if err != nil {
							b.Fatal(err)
						}
					}
					packet, ok := decoded.(*UDPPacket)
					if !ok {
						b.Fatalf("decoded message type %T, want *UDPPacket", decoded)
					}
					udpBenchmarkValidatePacket(b, packet, tc.packet)
					udpBenchmarkMessageSink = decoded
				})
			}
		}
	}
}
