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
	"bytes"
	"net"
	"testing"
)

func TestDatagramFrameRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		proxy   string
		addr    string
		payload []byte
	}{
		{"empty payload", "ocservudp", "1.2.3.4:5678", []byte{}},
		{"small", "p", "10.0.0.1:53", []byte("hello")},
		{"binary", "some-proxy-name", "192.168.1.1:1194", []byte{0x00, 0xff, 0x10, 0x00}},
		{"ipv6", "x", "[fe80::1]:9999", bytes.Repeat([]byte{0xab}, 1100)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raddr, err := net.ResolveUDPAddr("udp", c.addr)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			frame, err := encodeDgramFrame(c.proxy, raddr, c.payload)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			name, gotAddr, payload, err := decodeDgramFrame(frame)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if name != c.proxy {
				t.Errorf("name = %q, want %q", name, c.proxy)
			}
			if gotAddr.String() != raddr.String() {
				t.Errorf("addr = %s, want %s", gotAddr, raddr)
			}
			if !bytes.Equal(payload, c.payload) {
				t.Errorf("payload = %v, want %v", payload, c.payload)
			}
		})
	}
}

func TestDatagramFrameEncodeErrors(t *testing.T) {
	if _, err := encodeDgramFrame("p", nil, []byte("x")); err == nil {
		t.Error("expected error for nil remote address")
	}
	longName := string(bytes.Repeat([]byte{'a'}, 256))
	raddr, _ := net.ResolveUDPAddr("udp", "1.2.3.4:5")
	if _, err := encodeDgramFrame(longName, raddr, []byte("x")); err == nil {
		t.Error("expected error for oversized proxy name")
	}
}

func TestDatagramFrameDecodeRejectsMalformed(t *testing.T) {
	cases := [][]byte{
		nil,
		{0x00},                       // too short
		{0xF1, 0x05, 'a'},            // name length exceeds data
		{0x00, 0x01, 'a', 0x01, 'b'}, // wrong version
	}
	for i, data := range cases {
		if _, _, _, err := decodeDgramFrame(data); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}
