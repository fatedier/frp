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

//go:build !frps

package httpsserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func certificatePEM(t *testing.T, serial int64) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

func handshake(t *testing.T, cfg *tls.Config) (*tls.Conn, *tls.Conn) {
	t.Helper()
	clientRaw, serverRaw := net.Pipe()
	t.Cleanup(func() { _ = clientRaw.Close(); _ = serverRaw.Close() })
	require.NoError(t, clientRaw.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, serverRaw.SetDeadline(time.Now().Add(5*time.Second)))
	client := tls.Client(clientRaw, &tls.Config{InsecureSkipVerify: true})
	server := tls.Server(serverRaw, cfg)
	done := make(chan error, 1)
	go func() { done <- server.Handshake() }()
	require.NoError(t, client.Handshake())
	require.NoError(t, <-done)
	return client, server
}

func TestReloadCertificate(t *testing.T) {
	dir := t.TempDir()
	crtPath, keyPath := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	certPEM, keyPEM := certificatePEM(t, 1)
	require.NoError(t, os.WriteFile(crtPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))
	s, err := New(nil, crtPath, keyPath, nil)
	require.NoError(t, err)
	originalState := s.state.Load()
	original := originalState.cert
	oldClient, oldServer := handshake(t, s.TLSConfig)
	require.Equal(t, int64(1), oldClient.ConnectionState().PeerCertificates[0].SerialNumber.Int64())

	require.NoError(t, s.Reload())
	require.Same(t, originalState, s.state.Load(), "unchanged bytes must not parse or publish a new pair")
	require.NoError(t, s.ReloadError())

	for _, tc := range []struct {
		name, path string
		data       []byte
	}{
		{"missing certificate", crtPath, nil},
		{"missing key", keyPath, nil},
		{"invalid certificate", crtPath, []byte("invalid")},
		{"invalid key", keyPath, []byte("invalid")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.data == nil {
				require.NoError(t, os.Remove(tc.path))
			} else {
				require.NoError(t, os.WriteFile(tc.path, tc.data, 0o600))
			}
			require.Error(t, s.Reload())
			require.Error(t, s.ReloadError())
			require.Same(t, original, s.state.Load().cert)
			client, _ := handshake(t, s.TLSConfig)
			require.Equal(t, int64(1), client.ConnectionState().PeerCertificates[0].SerialNumber.Int64())
			require.NoError(t, os.WriteFile(crtPath, certPEM, 0o600))
			require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))
			require.NoError(t, s.Reload())
			require.NoError(t, s.ReloadError())
			require.Same(t, original, s.state.Load().cert)
		})
	}
	// Even a key-only byte change must be noticed, regardless of parsed identity.
	require.NoError(t, os.WriteFile(keyPath, append(keyPEM, '\n'), 0o600))
	require.NoError(t, s.Reload())
	require.NotSame(t, original, s.state.Load().cert)

	nextCert, nextKey := certificatePEM(t, 2)
	require.NoError(t, os.WriteFile(crtPath, nextCert, 0o600))
	beforeMismatch := s.state.Load().cert
	require.Error(t, s.Reload(), "mismatched pair must retain the active certificate")
	require.Same(t, beforeMismatch, s.state.Load().cert)
	require.NoError(t, os.WriteFile(keyPath, nextKey, 0o600))
	beforeReload, _ := handshake(t, s.TLSConfig)
	require.Equal(t, int64(1), beforeReload.ConnectionState().PeerCertificates[0].SerialNumber.Int64())
	require.NoError(t, s.Reload())
	newClient, _ := handshake(t, s.TLSConfig)
	require.Equal(t, int64(2), newClient.ConnectionState().PeerCertificates[0].SerialNumber.Int64())

	require.NoError(t, oldClient.SetDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, oldServer.SetDeadline(time.Now().Add(5*time.Second)))
	done := make(chan error, 1)
	go func() { _, err := oldServer.Write([]byte("still connected")); done <- err }()
	data := make([]byte, len("still connected"))
	_, err = io.ReadFull(oldClient, data)
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Equal(t, "still connected", string(data))
}

func TestReloadCertificateKeepsSelfSigned(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	for _, paths := range [][2]string{{"", ""}, {missing, ""}, {"", missing}} {
		s, err := New(nil, paths[0], paths[1], nil)
		require.NoError(t, err)
		original := &s.TLSConfig.Certificates[0]
		require.Nil(t, s.TLSConfig.GetCertificate)
		require.NoError(t, s.Reload())
		require.Same(t, original, &s.TLSConfig.Certificates[0])
	}
}
