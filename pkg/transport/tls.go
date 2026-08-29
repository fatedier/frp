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

package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

func newCustomTLSKeyPair(certfile, keyfile string) (*tls.Certificate, error) {
	tlsCert, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

func newRandomTLSKeyPair() (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Generate a random positive serial number with 128 bits of entropy.
	// RFC 5280 requires serial numbers to be positive integers (not zero).
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	// Ensure serial number is positive (not zero)
	if serialNumber.Sign() == 0 {
		serialNumber = big.NewInt(1)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour * 10),
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&key.PublicKey,
		key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

// Only support one ca file to add
func newCertPool(caPath string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	caCrt, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	if !pool.AppendCertsFromPEM(caCrt) {
		return nil, fmt.Errorf("failed to parse CA certificate from file %q: no valid PEM certificates found", caPath)
	}

	return pool, nil
}

// reloadableCertLoader holds the server TLS certificate/key pair (and optional
// client CA) in memory and reloads it from disk on demand (e.g. when frps
// receives SIGHUP). Handshakes serve the in-memory material directly via the
// GetCertificate/GetConfigForClient callbacks, so the hot path has zero extra
// cost and performs no disk access on its own.
type reloadableCertLoader struct {
	certPath string
	keyPath  string
	caPath   string

	// base holds the server tls.Config the loader was created for, so that
	// GetConfigForClient can return a config that preserves any other settings
	// (MinVersion, cipher suites, etc.) configured on it.
	base *tls.Config

	mu         sync.RWMutex
	cert       *tls.Certificate
	clientCAs  *x509.CertPool
	clientAuth tls.ClientAuthType
}

func newReloadableCertLoader(certPath, keyPath, caPath string) (*reloadableCertLoader, error) {
	l := &reloadableCertLoader{
		certPath: certPath,
		keyPath:  keyPath,
		caPath:   caPath,
	}
	if caPath != "" {
		l.clientAuth = tls.RequireAndVerifyClientCert
	}
	if err := l.reload(); err != nil {
		return nil, err
	}
	return l, nil
}

// reload reads the certificate material from disk and atomically replaces the
// in-memory copy. On failure it returns an error without mutating the currently
// loaded material, so a broken update does not break existing connections.
func (l *reloadableCertLoader) reload() error {
	cert, err := tls.LoadX509KeyPair(l.certPath, l.keyPath)
	if err != nil {
		return fmt.Errorf("load server x509 key pair from %q/%q: %w", l.certPath, l.keyPath, err)
	}
	var clientCAs *x509.CertPool
	if l.caPath != "" {
		clientCAs, err = newCertPool(l.caPath)
		if err != nil {
			return err
		}
	}
	l.mu.Lock()
	l.cert = &cert
	l.clientCAs = clientCAs
	// Keep the seeded static fields in sync so consumers that read
	// base.Certificates directly also see the reloaded material. The primary
	// reload path for handshakes is the GetCertificate/GetConfigForClient
	// callbacks (which serve l.cert). base is only set after construction, so
	// it is nil during the initial reload and safely skipped there.
	if l.base != nil {
		l.base.Certificates = []tls.Certificate{*l.cert}
		if clientCAs != nil {
			l.base.ClientCAs = clientCAs
			l.base.ClientAuth = l.clientAuth
		}
	}
	l.mu.Unlock()
	return nil
}

// Reload re-reads the certificate material from disk. It is intended to be
// triggered by an external event such as a SIGHUP signal.
func (l *reloadableCertLoader) Reload() error {
	return l.reload()
}

func (l *reloadableCertLoader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.cert == nil {
		return nil, fmt.Errorf("no server certificate available")
	}
	return l.cert, nil
}

func (l *reloadableCertLoader) GetConfigForClient(*tls.ClientHelloInfo) (*tls.Config, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.cert == nil {
		return nil, fmt.Errorf("no server certificate available")
	}
	cfg := l.base.Clone()
	cfg.Certificates = []tls.Certificate{*l.cert}
	if l.clientCAs != nil {
		cfg.ClientCAs = l.clientCAs
		cfg.ClientAuth = l.clientAuth
	}
	// Clear the callback on the returned config: crypto/tls invokes
	// GetConfigForClient again on the result, so leaving it set would recurse
	// infinitely (stack overflow) on every handshake.
	cfg.GetConfigForClient = nil
	return cfg, nil
}

// NewServerTLSConfig builds a server tls.Config. When a custom certificate is
// configured it is loaded once into memory; callers that need hot-reload should
// use NewServerTLSConfigWithReloader and trigger Reload (e.g. on SIGHUP).
func NewServerTLSConfig(certPath, keyPath, caPath string) (*tls.Config, error) {
	cfg, _, err := NewServerTLSConfigWithReloader(certPath, keyPath, caPath)
	return cfg, err
}

// NewServerTLSConfigWithReloader is like NewServerTLSConfig but additionally
// returns a Reload function. Call it (for example in response to a SIGHUP
// signal) to re-read the certificate material from disk and apply it to new TLS
// handshakes without restarting the process.
func NewServerTLSConfigWithReloader(certPath, keyPath, caPath string) (*tls.Config, func() error, error) {
	base := &tls.Config{}

	if certPath == "" || keyPath == "" {
		// server will generate tls conf by itself; this is an in-memory random
		// certificate that cannot be hot-reloaded from disk.
		cert, err := newRandomTLSKeyPair()
		if err != nil {
			return nil, nil, err
		}
		base.Certificates = []tls.Certificate{*cert}
		// Auto-generated, in-memory certificate: it cannot be reloaded from
		// disk, so report no reloader. Callers (e.g. the SIGHUP handler) treat
		// a nil reloader as "nothing to reload" and skip it.
		return base, nil, nil
	}

	loader, err := newReloadableCertLoader(certPath, keyPath, caPath)
	if err != nil {
		return nil, nil, err
	}
	loader.base = base
	base.GetCertificate = loader.GetCertificate
	base.GetConfigForClient = loader.GetConfigForClient
	// Seed the static fields as well so non-callback consumers (e.g. QUIC)
	// have a valid initial certificate.
	base.Certificates = []tls.Certificate{*loader.cert}
	if loader.clientCAs != nil {
		base.ClientAuth = tls.RequireAndVerifyClientCert
		base.ClientCAs = loader.clientCAs
	}
	return base, loader.Reload, nil
}

func NewClientTLSConfig(certPath, keyPath, caPath, serverName string) (*tls.Config, error) {
	base := &tls.Config{}

	if certPath != "" && keyPath != "" {
		cert, err := newCustomTLSKeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}

		base.Certificates = []tls.Certificate{*cert}
	}

	base.ServerName = serverName

	if caPath != "" {
		pool, err := newCertPool(caPath)
		if err != nil {
			return nil, err
		}

		base.RootCAs = pool
		base.InsecureSkipVerify = false
	} else {
		base.InsecureSkipVerify = true
	}

	return base, nil
}

func NewRandomPrivateKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return keyPEM, nil
}
