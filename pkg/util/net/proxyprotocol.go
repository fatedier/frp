// Copyright 2025 The frp Authors
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

package net

import (
	"bytes"
	"fmt"
	"net"

	pp "github.com/pires/go-proxyproto"
)

func BuildProxyProtocolHeaderStruct(srcAddr, dstAddr net.Addr, version string) *pp.Header {
	var versionByte byte
	if version == "v1" {
		versionByte = 1
	} else {
		versionByte = 2 // default to v2
	}
	return pp.HeaderProxyFromAddrs(versionByte, srcAddr, dstAddr)
}

func BuildProxyProtocolHeader(srcAddr, dstAddr net.Addr, version string) ([]byte, error) {
	h := BuildProxyProtocolHeaderStruct(srcAddr, dstAddr, version)

	// Convert header to bytes using a buffer
	var buf bytes.Buffer
	_, err := h.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write proxy protocol header: %v", err)
	}
	return buf.Bytes(), nil
}
