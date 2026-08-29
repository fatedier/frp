package transport

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func writeRandomCertKey(t *testing.T, dir, certName, keyName string) {
	t.Helper()
	cert, err := newRandomTLSKeyPair()
	if err != nil {
		t.Fatalf("newRandomTLSKeyPair: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	key, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("unexpected private key type")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(filepath.Join(dir, certName), certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, keyName), keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
}

func TestServerTLSConfigHotReload(t *testing.T) {
	dir, err := os.MkdirTemp("", "frp-tls-reload")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	writeRandomCertKey(t, dir, "cert.pem", "key.pem")

	cfg, reload, err := NewServerTLSConfigWithReloader(certPath, keyPath, "")
	if err != nil {
		t.Fatal(err)
	}

	first, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	firstDER := first.Certificate[0]

	writeRandomCertKey(t, dir, "cert_new.pem", "key_new.pem")
	if err := os.Rename(filepath.Join(dir, "cert_new.pem"), certPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "key_new.pem"), keyPath); err != nil {
		t.Fatal(err)
	}

	if err := reload(); err != nil {
		t.Fatal(err)
	}

	second, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	secondDER := second.Certificate[0]

	if string(firstDER) == string(secondDER) {
		t.Fatal("certificate was not reloaded after calling Reload")
	}
}

func TestServerTLSConfigReloadPreservesUnchanged(t *testing.T) {
	dir, err := os.MkdirTemp("", "frp-tls-reload-unchanged")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	writeRandomCertKey(t, dir, "cert.pem", "key.pem")

	cfg, reload, err := NewServerTLSConfigWithReloader(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), "")
	if err != nil {
		t.Fatal(err)
	}

	first, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := reload(); err != nil {
		t.Fatal(err)
	}
	second, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.Certificate[0]) != string(second.Certificate[0]) {
		t.Fatal("certificate changed unexpectedly when the file was unchanged")
	}
}

func TestServerTLSConfigReloadGetConfigForClient(t *testing.T) {
	dir, err := os.MkdirTemp("", "frp-tls-reload-gcc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	writeRandomCertKey(t, dir, "cert.pem", "key.pem")

	cfg, reload, err := NewServerTLSConfigWithReloader(certPath, keyPath, "")
	if err != nil {
		t.Fatal(err)
	}

	gcc, err := cfg.GetConfigForClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	if gcc == nil {
		t.Fatal("GetConfigForClient returned nil config")
	}
	if gcc.GetConfigForClient != nil {
		t.Fatal("GetConfigForClient must clear its own callback to avoid infinite recursion")
	}
	first, err := gcc.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	firstDER := first.Certificate[0]

	writeRandomCertKey(t, dir, "cert_new.pem", "key_new.pem")
	if err := os.Rename(filepath.Join(dir, "cert_new.pem"), certPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "key_new.pem"), keyPath); err != nil {
		t.Fatal(err)
	}
	if err := reload(); err != nil {
		t.Fatal(err)
	}

	gcc2, err := cfg.GetConfigForClient(nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := gcc2.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstDER) == string(second.Certificate[0]) {
		t.Fatal("certificate was not reloaded through GetConfigForClient")
	}
}

func TestServerTLSConfigReloadNilForAutoGenCert(t *testing.T) {
	// With no cert/key configured the server generates an in-memory cert that
	// cannot be reloaded from disk, so the reloader must be nil.
	cfg, reload, err := NewServerTLSConfigWithReloader("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if reload != nil {
		t.Fatal("expected nil reloader for auto-generated certificate")
	}
	if cfg == nil {
		t.Fatal("expected a valid base config")
	}
}
