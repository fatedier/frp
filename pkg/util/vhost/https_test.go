package vhost

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetHTTPSHostname(t *testing.T) {
	require := require.New(t)

	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(err)
	defer l.Close()

	connCh := make(chan net.Conn, 1)
	acceptErrCh := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			acceptErrCh <- err
			return
		}
		connCh <- conn
	}()

	clientErrCh := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		conn, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "example.com",
		})
		if conn != nil {
			_ = conn.Close()
		}
		clientErrCh <- err
	}()

	var conn net.Conn
	select {
	case conn = <-connCh:
	case err := <-acceptErrCh:
		require.NoError(err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for accepted connection")
	}
	require.NotNil(conn)

	serverConn, infos, err := GetHTTPSHostname(conn)
	if serverConn != nil {
		_ = serverConn.Close()
	} else {
		_ = conn.Close()
	}
	require.NoError(err)
	require.Equal("example.com", infos["Host"])
	require.Equal("https", infos["Scheme"])

	select {
	case <-clientErrCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TLS client")
	}
}
