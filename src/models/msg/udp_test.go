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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	content string = "udp packet test"
	src     string = "1.1.1.1:1000"
	dst     string = "2.2.2.2:2000"

	udpMsg *UdpPacket
)

func init() {
	srcAddr, _ := net.ResolveUDPAddr("udp", src)
	dstAddr, _ := net.ResolveUDPAddr("udp", dst)
	udpMsg = NewUdpPacket([]byte(content), srcAddr, dstAddr)
}

func TestPack(t *testing.T) {
	assert := assert.New(t)
	msg := udpMsg.Pack()
	assert.Equal(string(msg), `{"content":"dWRwIHBhY2tldCB0ZXN0","src":"1.1.1.1:1000","dst":"2.2.2.2:2000"}`)
}

func TestUnpack(t *testing.T) {
	assert := assert.New(t)
	udpMsg.UnPack([]byte(`{"content":"dWRwIHBhY2tldCB0ZXN0","src":"1.1.1.1:1000","dst":"2.2.2.2:2000"}`))
	assert.Equal(content, string(udpMsg.Content))
	assert.Equal(src, udpMsg.Src.String())
	assert.Equal(dst, udpMsg.Dst.String())
}
