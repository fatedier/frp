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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"

	pp "github.com/pires/go-proxyproto"
)

type key string

const (
	ProxyProtocolKey key = "proxy_protocol"
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

// CheckProxyProtocol checks if the connection is using Proxy Protocol.
func CheckProxyProtocol(c net.Conn) (net.Conn, *pp.Header) {
	// Use a buffered reader so that bytes read during detection are not lost.
	// bufio.Reader will first return any buffered bytes, then continue reading from the underlying conn.
	reader := bufio.NewReader(c)

	// Try to read a proxy protocol header from the buffered reader.
	// If present, header will be returned and reader will have consumed the header bytes.
	hdr, err := pp.Read(reader)

	// Build a ReadWriteCloser that uses the buffered reader for Read and the original conn for Write/Close.
	rwc := struct {
		io.Reader
		io.Writer
		io.Closer
	}{
		Reader: reader,
		Writer: c,
		Closer: c,
	}

	// Wrap to net.Conn so callers can use it transparently.
	wrapped := WrapReadWriteCloserToConn(rwc, c)

	if err == nil && hdr != nil {
		return wrapped, hdr
	}

	// Not a proxy protocol (or error parsing). Return wrapped conn and nil header.
	return wrapped, nil
}

func ProxyProtocolFromContext(ctx context.Context) *pp.Header {
	if hdr, ok := ctx.Value(ProxyProtocolKey).(*pp.Header); ok {
		return hdr
	}
	return nil
}
