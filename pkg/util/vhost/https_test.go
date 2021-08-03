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

	l, err := net.Listen("tcp", ":")
	require.NoError(err)
	defer l.Close()

	var conn net.Conn
	go func() {
		conn, _ = l.Accept()
		require.NotNil(conn)
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		tls.Dial("tcp", l.Addr().String(), &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         "example.com",
		})
	}()

	time.Sleep(200 * time.Millisecond)
	_, infos, err := GetHTTPSHostname(conn)
	require.NoError(err)
	require.Equal("example.com", infos["Host"])
	require.Equal("https", infos["Scheme"])
}
