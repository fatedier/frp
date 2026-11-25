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
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newSelfSignedCert(t *testing.T, name string) (tls.Certificate, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		Subject:               pkix.Name{CommonName: name},
		DNSNames:              []string{name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
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
	_, systemPEM := newSelfSignedCert(t, "system.example.com")
	_, filePEM := newSelfSignedCert(t, "file.example.com")
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, filePEM, 0o600))

	systemPool := x509.NewCertPool()
	require.True(t, systemPool.AppendCertsFromPEM(systemPEM))
	filePool := x509.NewCertPool()
	require.True(t, filePool.AppendCertsFromPEM(filePEM))
	combinedPool := systemPool.Clone()
	require.True(t, combinedPool.AppendCertsFromPEM(filePEM))

	for _, test := range []struct {
		name               string
		caPath             string
		useSystemCa        bool
		insecureSkipVerify bool
		wantPool           *x509.CertPool
	}{
		{name: "no trust source", insecureSkipVerify: true},
		{name: "file CA only", caPath: caFile, wantPool: filePool},
		{name: "system CA only", useSystemCa: true},
		{name: "system and file CAs", caPath: caFile, useSystemCa: true, wantPool: combinedPool},
	} {
		t.Run(test.name, func(t *testing.T) {
			var loadedPool *x509.CertPool
			loads := 0
			loadSystemPool := func() (*x509.CertPool, error) {
				loads++
				loadedPool = systemPool.Clone()
				return loadedPool, nil
			}
			cfg, err := newClientTLSConfig("", "", test.caPath, "example.com", test.useSystemCa, loadSystemPool)
			require.NoError(t, err)
			require.Equal(t, "example.com", cfg.ServerName)
			require.Equal(t, test.insecureSkipVerify, cfg.InsecureSkipVerify)
			if test.wantPool == nil {
				require.Nil(t, cfg.RootCAs)
			} else {
				require.NotNil(t, cfg.RootCAs)
				require.True(t, cfg.RootCAs.Equal(test.wantPool))
			}
			if test.caPath != "" && test.useSystemCa {
				require.Equal(t, 1, loads)
				require.Same(t, loadedPool, cfg.RootCAs, "append to the returned system pool")
			} else {
				require.Zero(t, loads, "only combined trust needs an explicit system pool")
			}
		})
	}

	// The four-argument API keeps the original skip/file-only semantics for
	// xtcp visitors and plugins, regardless of the new transport option.
	cfg, err := NewClientTLSConfig("", "", "", "example.com")
	require.NoError(t, err)
	require.True(t, cfg.InsecureSkipVerify)
	require.Nil(t, cfg.RootCAs)
	cfg, err = NewClientTLSConfig("", "", caFile, "example.com")
	require.NoError(t, err)
	require.False(t, cfg.InsecureSkipVerify)
	require.True(t, cfg.RootCAs.Equal(filePool))
}

func TestNewClientTLSConfigCAErrors(t *testing.T) {
	_, caPEM := newSelfSignedCert(t, "localhost")
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.crt")
	badFile := filepath.Join(dir, "bad.crt")
	require.NoError(t, os.WriteFile(caFile, caPEM, 0o600))
	require.NoError(t, os.WriteFile(badFile, []byte("not a PEM certificate"), 0o600))
	systemErr := errors.New("system trust store unavailable")

	for _, test := range []struct {
		name        string
		caPath      string
		useSystemCa bool
		pool        *x509.CertPool
		poolErr     error
		wantErr     string
	}{
		{name: "file missing", caPath: filepath.Join(dir, "missing.crt"), wantErr: "missing.crt"},
		{name: "file invalid", caPath: badFile, wantErr: "no valid PEM certificates found"},
		{
			name: "combined file missing", caPath: filepath.Join(dir, "missing.crt"), useSystemCa: true,
			pool: x509.NewCertPool(), wantErr: "missing.crt",
		},
		{
			name: "combined file invalid", caPath: badFile, useSystemCa: true,
			pool: x509.NewCertPool(), wantErr: "no valid PEM certificates found",
		},
		{
			name: "system pool error", caPath: caFile, useSystemCa: true,
			pool: x509.NewCertPool(), poolErr: systemErr, wantErr: "failed to load system CA pool",
		},
		{name: "nil system pool", caPath: caFile, useSystemCa: true, wantErr: "no pool returned"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := newClientTLSConfig("", "", test.caPath, "localhost", test.useSystemCa,
				func() (*x509.CertPool, error) { return test.pool, test.poolErr })
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, cfg, "do not return a partially configured TLS client or fall back")
			if test.poolErr != nil {
				require.ErrorIs(t, err, test.poolErr)
			}
		})
	}
}

func TestClientTLSConfigHandshake(t *testing.T) {
	systemCert, systemPEM := newSelfSignedCert(t, "localhost")
	fileCert, filePEM := newSelfSignedCert(t, "localhost")
	untrustedCert, _ := newSelfSignedCert(t, "localhost")
	caFile := filepath.Join(t.TempDir(), "ca.crt")
	require.NoError(t, os.WriteFile(caFile, filePEM, 0o600))
	systemPool := x509.NewCertPool()
	require.True(t, systemPool.AppendCertsFromPEM(systemPEM))

	for _, server := range []struct {
		name string
		cert tls.Certificate
	}{
		{name: "system CA", cert: systemCert},
		{name: "file CA", cert: fileCert},
		{name: "untrusted CA", cert: untrustedCert},
	} {
		t.Run(server.name, func(t *testing.T) {
			ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{server.cert}})
			require.NoError(t, err)
			t.Cleanup(func() { _ = ln.Close() })
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
					_ = conn.(*tls.Conn).Handshake()
					_ = conn.Close()
				}
			}()

			for _, test := range []struct {
				name        string
				caPath      string
				useSystemCa bool
				serverName  string
				wantErr     bool
			}{
				{name: "no trust source"},
				{name: "file CA only", caPath: caFile, wantErr: server.name != "file CA"},
				// All certificates were just generated and are absent from the real
				// host store, even on hosts without any installed system roots.
				{name: "system CA only rejects unknown certificate", useSystemCa: true, wantErr: true},
				{name: "combined CAs", caPath: caFile, useSystemCa: true, wantErr: server.name == "untrusted CA"},
				{
					name: "combined CAs reject wrong name", caPath: caFile, useSystemCa: true,
					serverName: "other.example.com", wantErr: true,
				},
			} {
				t.Run(test.name, func(t *testing.T) {
					serverName := test.serverName
					if serverName == "" {
						serverName = "localhost"
					}
					var cfg *tls.Config
					var err error
					if test.caPath != "" && test.useSystemCa {
						cfg, err = newClientTLSConfig("", "", test.caPath, serverName, true,
							func() (*x509.CertPool, error) { return systemPool.Clone(), nil })
					} else {
						cfg, err = NewClientTLSConfigWithSystemCA("", "", test.caPath, serverName, test.useSystemCa)
					}
					require.NoError(t, err)
					conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", ln.Addr().String(), cfg)
					if conn != nil {
						_ = conn.Close()
					}
					if test.wantErr {
						var verificationErr *tls.CertificateVerificationError
						require.ErrorAs(t, err, &verificationErr)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	}
}
