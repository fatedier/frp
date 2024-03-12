// Copyright 2023 The frp Authors
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

package nathole

import (
	"bytes"
	"fmt"
	"net"
	"strconv"

	"github.com/fatedier/golib/crypto"
	"github.com/pion/stun/v2"

	"github.com/fatedier/frp/pkg/msg"
)

func EncodeMessage(m msg.Message, key []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := msg.WriteMsg(buffer, m); err != nil {
		return nil, err
	}

	buf, err := crypto.Encode(buffer.Bytes(), key)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func DecodeMessageInto(data, key []byte, m msg.Message) error {
	buf, err := crypto.Decode(data, key)
	if err != nil {
		return err
	}

	return msg.ReadMsgInto(bytes.NewReader(buf), m)
}

type ChangedAddress struct {
	IP   net.IP
	Port int
}

func (s *ChangedAddress) GetFrom(m *stun.Message) error {
	a := (*stun.MappedAddress)(s)
	return a.GetFromAs(m, stun.AttrChangedAddress)
}

func (s *ChangedAddress) String() string {
	return net.JoinHostPort(s.IP.String(), strconv.Itoa(s.Port))
}

func ListAllLocalIPs() ([]net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

func ListLocalIPsForNatHole(max int) ([]string, error) {
	if max <= 0 {
		return nil, fmt.Errorf("max must be greater than 0")
	}

	ips, err := ListAllLocalIPs()
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, max)
	for _, ip := range ips {
		if len(filtered) >= max {
			break
		}

		// ignore ipv6 address
		if ip.To4() == nil {
			continue
		}
		// ignore localhost IP
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}

		filtered = append(filtered, ip.String())
	}
	return filtered, nil
}
