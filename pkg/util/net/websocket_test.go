package net

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// genTestCert returns a self-signed TLS certificate for use in tests.
func genTestCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("x509 key pair: %v", err)
	}
	return cert
}

func TestWebsocketListenerHandleConnWSS(t *testing.T) {
	cert := genTestCert(t)
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	// A throwaway listener keeps NewWebsocketListener's internal HTTP server
	// busy without interfering with the wss test below.
	dummy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("dummy listen: %v", err)
	}
	defer dummy.Close()
	wl := NewWebsocketListener(dummy, "/~!frp", "/api/message")
	defer wl.Close()

	// The actual wss endpoint: a plain TCP listener that, after manually
	// terminating TLS (mirroring frps' CheckAndEnableTLS), hands the
	// plaintext connection to HandleConn.
	tlsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	defer tlsLn.Close()

	go func() {
		conn, err := tlsLn.Accept()
		if err != nil {
			return
		}
		tlsConn := tls.Server(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		_ = wl.HandleConn(tlsConn)
	}()

	// Client connects via wss to /api/message.
	origin := "https://" + tlsLn.Addr().String()
	cfg, err := websocket.NewConfig("wss://"+tlsLn.Addr().String()+"/api/message", origin)
	if err != nil {
		t.Fatalf("ws config: %v", err)
	}
	cfg.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := websocket.DialConfig(cfg)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer client.Close()

	// Server should accept the upgraded connection through Accept().
	srvConn, err := wl.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer srvConn.Close()

	// Regression: RemoteAddr() must report the real client TCP address, not
	// the websocket origin URL (e.g. "http://ssps.sazas.cn:1000").
	addr := srvConn.RemoteAddr().String()
	if addr == "" {
		t.Fatalf("RemoteAddr() returned empty string")
	}
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("RemoteAddr() = %q, want the real client address (127.0.0.1:*)", addr)
	}
	if strings.Contains(addr, "ssps.sazas.cn") {
		t.Fatalf("RemoteAddr() returned the websocket origin URL: %q", addr)
	}

	// Exchange a payload to prove the tunnel works end to end.
	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("client write: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := srvConn.Read(buf); err != nil {
		t.Fatalf("server read: %v", err)
	}
	if string(buf) != "hello" {
		t.Fatalf("unexpected payload: %q", string(buf))
	}
}

func TestTryWebsocket(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Simulate a client that immediately sends a websocket upgrade request.
	go func() {
		_, _ = client.Write([]byte("GET /api/message HTTP/1.1\r\nUpgrade: websocket\r\n\r\n"))
	}()

	restored, ok := TryWebsocket(server, []string{"/~!frp", "/api/message"})
	if !ok {
		t.Fatalf("TryWebsocket should detect the websocket upgrade")
	}
	// The peeked bytes must be preserved so the websocket handshake can
	// continue using the restored connection.
	buf := make([]byte, 64)
	n, err := restored.Read(buf)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if !containsBytes(buf[:n], []byte("GET /api/message")) {
		t.Fatalf("peeked bytes not preserved: %q", string(buf[:n]))
	}
}

func containsBytes(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}
