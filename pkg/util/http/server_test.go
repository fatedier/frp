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

package http

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestNewServerDoesNotLeakListenerOnTLSFailure(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, []byte("invalid certificate"), 0o600))
	require.NoError(t, os.WriteFile(keyFile, []byte("invalid key"), 0o600))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := ln.Addr().String()
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	svr, err := NewServer(v1.WebServerConfig{
		Addr: "127.0.0.1",
		Port: port,
		TLS:  &v1.TLSConfig{CertFile: certFile, KeyFile: keyFile},
	})
	if svr != nil {
		defer svr.Close()
	}
	require.ErrorContains(t, err, "failed to find any PEM data in certificate input")
	require.Nil(t, svr)

	ln, err = net.Listen("tcp", address)
	require.NoError(t, err, "web server port was not released")
	require.NoError(t, ln.Close())
}
