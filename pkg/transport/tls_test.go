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

package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newSelfSignedCert(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return cert, certPEM
}

func TestNewClientTLSConfigVerification(t *testing.T) {
	_, caPEM := newSelfSignedCert(t)
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, caPEM, 0o600))

	tests := []struct {
		name               string
		caPath             string
		useSystemRoots     bool
		insecureSkipVerify bool
		rootCAsSet         bool
	}{
		{
			name:               "no trusted ca file, no system roots",
			insecureSkipVerify: true,
		},
		{
			name:           "system roots",
			useSystemRoots: true,
		},
		{
			name:       "trusted ca file",
			caPath:     caFile,
			rootCAsSet: true,
		},
		{
			name:           "trusted ca file takes precedence over system roots",
			caPath:         caFile,
			useSystemRoots: true,
			rootCAsSet:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			cfg, err := NewClientTLSConfigWithSystemRoots("", "", test.caPath, "example.com", test.useSystemRoots)
			require.NoError(err)
			require.Equal("example.com", cfg.ServerName)
			require.Equal(test.insecureSkipVerify, cfg.InsecureSkipVerify)
			require.Equal(test.rootCAsSet, cfg.RootCAs != nil)
		})
	}
}

func TestClientTLSConfigHandshakeWithUntrustedServer(t *testing.T) {
	require := require.New(t)

	cert, _ := newSelfSignedCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	require.NoError(err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.(*tls.Conn).Handshake()
			_ = conn.Close()
		}
	}()

	dial := func(useSystemRoots bool) error {
		cfg, err := NewClientTLSConfigWithSystemRoots("", "", "", "localhost", useSystemRoots)
		require.NoError(err)
		conn, err := tls.Dial("tcp", ln.Addr().String(), cfg)
		if err != nil {
			return err
		}
		return conn.Close()
	}

	require.NoError(dial(false))

	err = dial(true)
	require.Error(err)
	var verificationErr *tls.CertificateVerificationError
	require.True(errors.As(err, &verificationErr), "expected a certificate verification error, got %v", err)
}
