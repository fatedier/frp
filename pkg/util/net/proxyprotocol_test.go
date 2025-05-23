package net

import (
	"net"
	"testing"

	pp "github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/require"
)

func TestBuildProxyProtocolHeader(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name        string
		srcAddr     net.Addr
		dstAddr     net.Addr
		version     string
		expectError bool
	}{
		{
			name:        "UDP IPv4 v2",
			srcAddr:     &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			dstAddr:     &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 3306},
			version:     "v2",
			expectError: false,
		},
		{
			name:        "TCP IPv4 v1",
			srcAddr:     &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			dstAddr:     &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 80},
			version:     "v1",
			expectError: false,
		},
		{
			name:        "UDP IPv6 v2",
			srcAddr:     &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 12345},
			dstAddr:     &net.UDPAddr{IP: net.ParseIP("::1"), Port: 3306},
			version:     "v2",
			expectError: false,
		},
		{
			name:        "TCP IPv6 v1",
			srcAddr:     &net.TCPAddr{IP: net.ParseIP("::1"), Port: 12345},
			dstAddr:     &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 80},
			version:     "v1",
			expectError: false,
		},
		{
			name:        "nil source address",
			srcAddr:     nil,
			dstAddr:     &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 3306},
			version:     "v2",
			expectError: false,
		},
		{
			name:        "nil destination address",
			srcAddr:     &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			dstAddr:     nil,
			version:     "v2",
			expectError: false,
		},
		{
			name:        "unsupported address type",
			srcAddr:     &net.UnixAddr{Name: "/tmp/test.sock", Net: "unix"},
			dstAddr:     &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 3306},
			version:     "v2",
			expectError: false,
		},
	}

	for _, tt := range tests {
		header, err := BuildProxyProtocolHeader(tt.srcAddr, tt.dstAddr, tt.version)

		if tt.expectError {
			require.Error(err, "test case: %s", tt.name)
			continue
		}

		require.NoError(err, "test case: %s", tt.name)
		require.NotEmpty(header, "test case: %s", tt.name)
	}
}

func TestBuildProxyProtocolHeaderStruct(t *testing.T) {
	require := require.New(t)

	tests := []struct {
		name               string
		srcAddr            net.Addr
		dstAddr            net.Addr
		version            string
		expectedProtocol   pp.AddressFamilyAndProtocol
		expectedVersion    byte
		expectedCommand    pp.ProtocolVersionAndCommand
		expectedSourceAddr net.Addr
		expectedDestAddr   net.Addr
	}{
		{
			name:               "TCP IPv4 v2",
			srcAddr:            &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			dstAddr:            &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 80},
			version:            "v2",
			expectedProtocol:   pp.TCPv4,
			expectedVersion:    2,
			expectedCommand:    pp.PROXY,
			expectedSourceAddr: &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			expectedDestAddr:   &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 80},
		},
		{
			name:               "UDP IPv6 v1",
			srcAddr:            &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 12345},
			dstAddr:            &net.UDPAddr{IP: net.ParseIP("::1"), Port: 3306},
			version:            "v1",
			expectedProtocol:   pp.UDPv6,
			expectedVersion:    1,
			expectedCommand:    pp.PROXY,
			expectedSourceAddr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 12345},
			expectedDestAddr:   &net.UDPAddr{IP: net.ParseIP("::1"), Port: 3306},
		},
		{
			name:               "TCP IPv6 default version",
			srcAddr:            &net.TCPAddr{IP: net.ParseIP("::1"), Port: 12345},
			dstAddr:            &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 80},
			version:            "",
			expectedProtocol:   pp.TCPv6,
			expectedVersion:    2, // default to v2
			expectedCommand:    pp.PROXY,
			expectedSourceAddr: &net.TCPAddr{IP: net.ParseIP("::1"), Port: 12345},
			expectedDestAddr:   &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 80},
		},
		{
			name:               "nil source address",
			srcAddr:            nil,
			dstAddr:            &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 3306},
			version:            "v2",
			expectedProtocol:   pp.UNSPEC,
			expectedVersion:    2,
			expectedCommand:    pp.LOCAL,
			expectedSourceAddr: nil, // go-proxyproto sets both to nil when srcAddr is nil
			expectedDestAddr:   nil,
		},
		{
			name:               "nil destination address",
			srcAddr:            &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345},
			dstAddr:            nil,
			version:            "v2",
			expectedProtocol:   pp.UNSPEC,
			expectedVersion:    2,
			expectedCommand:    pp.LOCAL,
			expectedSourceAddr: nil, // go-proxyproto sets both to nil when dstAddr is nil
			expectedDestAddr:   nil,
		},
		{
			name:               "unsupported address type",
			srcAddr:            &net.UnixAddr{Name: "/tmp/test.sock", Net: "unix"},
			dstAddr:            &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 3306},
			version:            "v2",
			expectedProtocol:   pp.UNSPEC,
			expectedVersion:    2,
			expectedCommand:    pp.LOCAL,
			expectedSourceAddr: nil, // go-proxyproto sets both to nil for unsupported types
			expectedDestAddr:   nil,
		},
	}

	for _, tt := range tests {
		header := BuildProxyProtocolHeaderStruct(tt.srcAddr, tt.dstAddr, tt.version)

		require.NotNil(header, "test case: %s", tt.name)

		require.Equal(tt.expectedCommand, header.Command, "test case: %s", tt.name)
		require.Equal(tt.expectedSourceAddr, header.SourceAddr, "test case: %s", tt.name)
		require.Equal(tt.expectedDestAddr, header.DestinationAddr, "test case: %s", tt.name)
		require.Equal(tt.expectedProtocol, header.TransportProtocol, "test case: %s", tt.name)
		require.Equal(tt.expectedVersion, header.Version, "test case: %s", tt.name)
	}
}
