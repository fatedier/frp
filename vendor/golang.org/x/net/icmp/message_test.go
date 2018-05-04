// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icmp_test

import (
	"net"
	"reflect"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/internal/iana"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestMarshalAndParseMessage(t *testing.T) {
	fn := func(t *testing.T, proto int, tms []icmp.Message) {
		var pshs [][]byte
		switch proto {
		case iana.ProtocolICMP:
			pshs = [][]byte{nil}
		case iana.ProtocolIPv6ICMP:
			pshs = [][]byte{
				icmp.IPv6PseudoHeader(net.ParseIP("fe80::1"), net.ParseIP("ff02::1")),
				nil,
			}
		}
		for i, tm := range tms {
			for _, psh := range pshs {
				b, err := tm.Marshal(psh)
				if err != nil {
					t.Fatal(err)
				}
				m, err := icmp.ParseMessage(proto, b)
				if err != nil {
					t.Fatal(err)
				}
				if m.Type != tm.Type || m.Code != tm.Code {
					t.Errorf("#%d: got %#v; want %#v", i, m, &tm)
				}
				if !reflect.DeepEqual(m.Body, tm.Body) {
					t.Errorf("#%d: got %#v; want %#v", i, m.Body, tm.Body)
				}
			}
		}
	}

	t.Run("IPv4", func(t *testing.T) {
		fn(t, iana.ProtocolICMP,
			[]icmp.Message{
				{
					Type: ipv4.ICMPTypeDestinationUnreachable, Code: 15,
					Body: &icmp.DstUnreach{
						Data: []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv4.ICMPTypeTimeExceeded, Code: 1,
					Body: &icmp.TimeExceeded{
						Data: []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv4.ICMPTypeParameterProblem, Code: 2,
					Body: &icmp.ParamProb{
						Pointer: 8,
						Data:    []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv4.ICMPTypeEcho, Code: 0,
					Body: &icmp.Echo{
						ID: 1, Seq: 2,
						Data: []byte("HELLO-R-U-THERE"),
					},
				},
				{
					Type: ipv4.ICMPTypeExtendedEchoRequest, Code: 0,
					Body: &icmp.ExtendedEchoRequest{
						ID: 1, Seq: 2,
					},
				},
				{
					Type: ipv4.ICMPTypeExtendedEchoReply, Code: 0,
					Body: &icmp.ExtendedEchoReply{
						State: 4 /* Delay */, Active: true, IPv4: true,
					},
				},
				{
					Type: ipv4.ICMPTypePhoturis,
					Body: &icmp.DefaultMessageBody{
						Data: []byte{0x80, 0x40, 0x20, 0x10},
					},
				},
			})
	})
	t.Run("IPv6", func(t *testing.T) {
		fn(t, iana.ProtocolIPv6ICMP,
			[]icmp.Message{
				{
					Type: ipv6.ICMPTypeDestinationUnreachable, Code: 6,
					Body: &icmp.DstUnreach{
						Data: []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv6.ICMPTypePacketTooBig, Code: 0,
					Body: &icmp.PacketTooBig{
						MTU:  1<<16 - 1,
						Data: []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv6.ICMPTypeTimeExceeded, Code: 1,
					Body: &icmp.TimeExceeded{
						Data: []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv6.ICMPTypeParameterProblem, Code: 2,
					Body: &icmp.ParamProb{
						Pointer: 8,
						Data:    []byte("ERROR-INVOKING-PACKET"),
					},
				},
				{
					Type: ipv6.ICMPTypeEchoRequest, Code: 0,
					Body: &icmp.Echo{
						ID: 1, Seq: 2,
						Data: []byte("HELLO-R-U-THERE"),
					},
				},
				{
					Type: ipv6.ICMPTypeExtendedEchoRequest, Code: 0,
					Body: &icmp.ExtendedEchoRequest{
						ID: 1, Seq: 2,
					},
				},
				{
					Type: ipv6.ICMPTypeExtendedEchoReply, Code: 0,
					Body: &icmp.ExtendedEchoReply{
						State: 5 /* Probe */, Active: true, IPv6: true,
					},
				},
				{
					Type: ipv6.ICMPTypeDuplicateAddressConfirmation,
					Body: &icmp.DefaultMessageBody{
						Data: []byte{0x80, 0x40, 0x20, 0x10},
					},
				},
			})
	})
}
