// Copyright 2016 fatedier, fatedier@gmail.com
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
	"encoding/base64"
	"encoding/json"
	"net"
)

type UdpPacket struct {
	Content []byte       `json:"-"`
	Src     *net.UDPAddr `json:"-"`
	Dst     *net.UDPAddr `json:"-"`

	EncodeContent string `json:"content"`
	SrcStr        string `json:"src"`
	DstStr        string `json:"dst"`
}

func NewUdpPacket(content []byte, src, dst *net.UDPAddr) *UdpPacket {
	up := &UdpPacket{
		Src:           src,
		Dst:           dst,
		EncodeContent: base64.StdEncoding.EncodeToString(content),
		SrcStr:        src.String(),
		DstStr:        dst.String(),
	}
	return up
}

// parse one udp packet struct to bytes
func (up *UdpPacket) Pack() []byte {
	b, _ := json.Marshal(up)
	return b
}

// parse from bytes to UdpPacket struct
func (up *UdpPacket) UnPack(packet []byte) error {
	err := json.Unmarshal(packet, &up)
	if err != nil {
		return err
	}

	up.Content, err = base64.StdEncoding.DecodeString(up.EncodeContent)
	if err != nil {
		return err
	}

	up.Src, err = net.ResolveUDPAddr("udp", up.SrcStr)
	if err != nil {
		return err
	}

	up.Dst, err = net.ResolveUDPAddr("udp", up.DstStr)
	if err != nil {
		return err
	}
	return nil
}
