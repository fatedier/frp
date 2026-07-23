package nathole

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/fatedier/golib/net/stun"
	"github.com/stretchr/testify/require"
)

const (
	testBindingRequest  = 0x0001
	testBindingSuccess  = 0x0101
	testBindingError    = 0x0111
	testMagicCookie     = 0x2112a442
	testAttrMapped      = 0x0001
	testAttrChanged     = 0x0005
	testAttrErrorCode   = 0x0009
	testAttrXORMapped   = 0x0020
	testAttrOther       = 0x802c
	testSTUNHeaderSize  = 20
	testSTUNServerLimit = time.Second
)

type testSTUNAttribute struct {
	typ   uint16
	value []byte
}

type testSTUNExchange struct {
	source *net.UDPAddr
	err    error
}

func listenTestUDP4(t *testing.T) *net.UDPConn {
	t.Helper()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func serveOneSTUNRequest(
	server *net.UDPConn,
	buildResponse func([]byte, *net.UDPAddr) ([]byte, error),
) <-chan testSTUNExchange {
	done := make(chan testSTUNExchange, 1)
	go func() {
		if err := server.SetDeadline(time.Now().Add(testSTUNServerLimit)); err != nil {
			done <- testSTUNExchange{err: err}
			return
		}
		buffer := make([]byte, 1024)
		n, source, err := server.ReadFromUDP(buffer)
		if err == nil && buildResponse != nil {
			var response []byte
			response, err = buildResponse(buffer[:n], source)
			if err == nil && response != nil {
				_, err = server.WriteToUDP(response, source)
			}
		}
		done <- testSTUNExchange{source: source, err: err}
	}()
	return done
}

func waitSTUNExchange(t *testing.T, done <-chan testSTUNExchange) *net.UDPAddr {
	t.Helper()

	select {
	case exchange := <-done:
		require.NoError(t, exchange.err)
		return exchange.source
	case <-time.After(testSTUNServerLimit):
		t.Fatal("timed out waiting for local STUN server")
		return nil
	}
}

func makeTestSTUNResponse(request []byte, typ uint16, attributes ...testSTUNAttribute) ([]byte, error) {
	if len(request) != testSTUNHeaderSize || binary.BigEndian.Uint16(request[0:2]) != testBindingRequest ||
		binary.BigEndian.Uint32(request[4:8]) != testMagicCookie {
		return nil, fmt.Errorf("invalid Binding request")
	}

	length := 0
	for _, attribute := range attributes {
		length += 4 + (len(attribute.value)+3)&^3
	}
	response := make([]byte, testSTUNHeaderSize, testSTUNHeaderSize+length)
	binary.BigEndian.PutUint16(response[0:2], typ)
	binary.BigEndian.PutUint16(response[2:4], uint16(length))
	binary.BigEndian.PutUint32(response[4:8], testMagicCookie)
	copy(response[8:20], request[8:20])

	for _, attribute := range attributes {
		start := len(response)
		paddedLength := (len(attribute.value) + 3) &^ 3
		response = append(response, make([]byte, 4+paddedLength)...)
		binary.BigEndian.PutUint16(response[start:start+2], attribute.typ)
		binary.BigEndian.PutUint16(response[start+2:start+4], uint16(len(attribute.value)))
		copy(response[start+4:], attribute.value)
	}
	return response, nil
}

func testIPv4AddressValue(ip net.IP, port int, xor bool) []byte {
	value := make([]byte, 8)
	value[1] = 0x01
	binary.BigEndian.PutUint16(value[2:4], uint16(port))
	copy(value[4:], ip.To4())
	if xor {
		binary.BigEndian.PutUint16(value[2:4], binary.BigEndian.Uint16(value[2:4])^uint16(testMagicCookie>>16))
		for i := range 4 {
			value[4+i] ^= byte(uint32(testMagicCookie) >> uint(24-8*i))
		}
	}
	return value
}

func TestDiscoverReusesLocalPortAndPreservesNATClassification(t *testing.T) {
	tests := []struct {
		name             string
		secondMapped     string
		secondMappedPort int
		wantNATType      string
		wantBehavior     string
	}{
		{
			name:             "same mapped address",
			secondMapped:     "198.51.100.10:40000",
			secondMappedPort: 40000,
			wantNATType:      EasyNAT,
			wantBehavior:     BehaviorNoChange,
		},
		{
			name:             "different mapped port",
			secondMapped:     "198.51.100.10:40001",
			secondMappedPort: 40001,
			wantNATType:      HardNAT,
			wantBehavior:     BehaviorPortChanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := listenTestUDP4(t)
			alternate := listenTestUDP4(t)
			alternateAddr := alternate.LocalAddr().(*net.UDPAddr)

			primaryDone := serveOneSTUNRequest(primary, func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				return makeTestSTUNResponse(request, testBindingSuccess,
					testSTUNAttribute{typ: testAttrXORMapped, value: testIPv4AddressValue(net.ParseIP("198.51.100.10"), 40000, true)},
					testSTUNAttribute{typ: testAttrOther, value: testIPv4AddressValue(alternateAddr.IP, alternateAddr.Port, false)},
				)
			})
			alternateDone := serveOneSTUNRequest(alternate, func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				return makeTestSTUNResponse(request, testBindingSuccess,
					testSTUNAttribute{typ: testAttrXORMapped, value: testIPv4AddressValue(net.ParseIP("198.51.100.10"), tt.secondMappedPort, true)},
				)
			})

			addresses, localAddr, err := Discover([]string{primary.LocalAddr().String()}, "")
			require.NoError(t, err)
			require.Equal(t, []string{"198.51.100.10:40000", tt.secondMapped}, addresses)

			primarySource := waitSTUNExchange(t, primaryDone)
			alternateSource := waitSTUNExchange(t, alternateDone)
			require.Equal(t, primarySource.Port, alternateSource.Port)
			require.Equal(t, localAddr.(*net.UDPAddr).Port, primarySource.Port)

			feature, err := ClassifyNATFeature(addresses, nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantNATType, feature.NatType)
			require.Equal(t, tt.wantBehavior, feature.Behavior)
		})
	}
}

func TestDoSTUNRequestMapsLegacyAndModernAddresses(t *testing.T) {
	tests := []struct {
		name         string
		attributes   []testSTUNAttribute
		wantExternal string
		wantOther    string
	}{
		{
			name: "legacy",
			attributes: []testSTUNAttribute{
				{typ: testAttrMapped, value: testIPv4AddressValue(net.ParseIP("192.0.2.1"), 1000, false)},
				{typ: testAttrChanged, value: testIPv4AddressValue(net.ParseIP("192.0.2.2"), 2000, false)},
			},
			wantExternal: "192.0.2.1:1000",
			wantOther:    "192.0.2.2:2000",
		},
		{
			name: "modern takes precedence",
			attributes: []testSTUNAttribute{
				{typ: testAttrMapped, value: testIPv4AddressValue(net.ParseIP("192.0.2.1"), 1000, false)},
				{typ: testAttrXORMapped, value: testIPv4AddressValue(net.ParseIP("198.51.100.1"), 3000, true)},
				{typ: testAttrChanged, value: testIPv4AddressValue(net.ParseIP("192.0.2.2"), 2000, false)},
				{typ: testAttrOther, value: testIPv4AddressValue(net.ParseIP("198.51.100.2"), 4000, false)},
			},
			wantExternal: "198.51.100.1:3000",
			wantOther:    "198.51.100.2:4000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := listenTestUDP4(t)
			done := serveOneSTUNRequest(server, func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				return makeTestSTUNResponse(request, testBindingSuccess, tt.attributes...)
			})
			conn, err := listen("")
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			response, err := conn.doSTUNRequest(server.LocalAddr().String())
			require.NoError(t, err)
			require.Equal(t, tt.wantExternal, response.externalAddr)
			require.Equal(t, tt.wantOther, response.otherAddr)
			waitSTUNExchange(t, done)
		})
	}
}

func TestSTUNResponseErrorsAndMissingAddresses(t *testing.T) {
	tests := []struct {
		name          string
		buildResponse func([]byte, *net.UDPAddr) ([]byte, error)
		request       func(*discoverConn, string) error
		checkError    func(*testing.T, error)
	}{
		{
			name: "correlated malformed response",
			buildResponse: func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				response, err := makeTestSTUNResponse(request, testBindingSuccess)
				if err == nil {
					binary.BigEndian.PutUint16(response[2:4], 4)
				}
				return response, err
			},
			request: func(conn *discoverConn, server string) error {
				_, err := conn.doSTUNRequest(server)
				return err
			},
			checkError: func(t *testing.T, err error) {
				require.ErrorIs(t, err, stun.ErrMalformedResponse)
			},
		},
		{
			name: "Binding error response",
			buildResponse: func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				return makeTestSTUNResponse(request, testBindingError, testSTUNAttribute{
					typ:   testAttrErrorCode,
					value: []byte{0, 0, 4, 20, 'U', 'n', 'k', 'n', 'o', 'w', 'n'},
				})
			},
			request: func(conn *discoverConn, server string) error {
				_, err := conn.doSTUNRequest(server)
				return err
			},
			checkError: func(t *testing.T, err error) {
				var responseErr *stun.ResponseError
				require.ErrorAs(t, err, &responseErr)
				require.Equal(t, 420, responseErr.Code)
			},
		},
		{
			name: "missing mapped address",
			buildResponse: func(request []byte, _ *net.UDPAddr) ([]byte, error) {
				return makeTestSTUNResponse(request, testBindingSuccess,
					testSTUNAttribute{typ: testAttrOther, value: testIPv4AddressValue(net.ParseIP("192.0.2.2"), 2000, false)},
				)
			},
			request: func(conn *discoverConn, server string) error {
				_, err := conn.discoverFromStunServer(server)
				return err
			},
			checkError: func(t *testing.T, err error) {
				require.EqualError(t, err, "no external address found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := listenTestUDP4(t)
			done := serveOneSTUNRequest(server, tt.buildResponse)
			conn, err := listen("")
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			err = tt.request(conn, server.LocalAddr().String())
			tt.checkError(t, err)
			waitSTUNExchange(t, done)
		})
	}

	t.Run("missing other address", func(t *testing.T) {
		server := listenTestUDP4(t)
		done := serveOneSTUNRequest(server, func(request []byte, _ *net.UDPAddr) ([]byte, error) {
			return makeTestSTUNResponse(request, testBindingSuccess,
				testSTUNAttribute{typ: testAttrXORMapped, value: testIPv4AddressValue(net.ParseIP("198.51.100.1"), 3000, true)},
			)
		})

		_, err := Prepare([]string{server.LocalAddr().String()}, PrepareOptions{})
		require.EqualError(t, err, "discover error: not enough addresses")
		waitSTUNExchange(t, done)
	})
}

func TestSTUNTimeoutUsesCallerDeadlineWithoutRetry(t *testing.T) {
	originalTimeout := responseTimeout
	responseTimeout = 50 * time.Millisecond
	t.Cleanup(func() { responseTimeout = originalTimeout })

	server := listenTestUDP4(t)
	done := serveOneSTUNRequest(server, nil)
	conn, err := listen("")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = conn.doSTUNRequest(server.LocalAddr().String())
	require.EqualError(t, err, "wait response from stun server timeout")
	waitSTUNExchange(t, done)

	require.NoError(t, server.SetReadDeadline(time.Now().Add(50*time.Millisecond)))
	_, _, err = server.ReadFromUDP(make([]byte, 1))
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
	require.True(t, netErr.Timeout())
}

func TestSTUNClientLeavesSocketAndDeadlineWithCaller(t *testing.T) {
	originalTimeout := responseTimeout
	responseTimeout = 100 * time.Millisecond
	t.Cleanup(func() { responseTimeout = originalTimeout })

	server := listenTestUDP4(t)
	unrelated := listenTestUDP4(t)
	done := serveOneSTUNRequest(server, func(request []byte, source *net.UDPAddr) ([]byte, error) {
		response, err := makeTestSTUNResponse(request, testBindingSuccess,
			testSTUNAttribute{typ: testAttrXORMapped, value: testIPv4AddressValue(net.ParseIP("198.51.100.5"), 5000, true)},
		)
		if err != nil {
			return nil, err
		}
		if _, err := unrelated.WriteToUDP(response, source); err != nil {
			return nil, err
		}
		return response, nil
	})
	conn, err := listen("")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	response, err := conn.doSTUNRequest(server.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "198.51.100.5:5000", response.externalAddr)
	waitSTUNExchange(t, done)

	_, _, err = conn.conn.ReadFromUDP(make([]byte, 1))
	var netErr net.Error
	require.True(t, errors.As(err, &netErr))
	require.True(t, netErr.Timeout())

	require.NoError(t, conn.conn.SetDeadline(time.Time{}))
	require.NoError(t, server.SetReadDeadline(time.Now().Add(testSTUNServerLimit)))
	_, err = conn.conn.WriteToUDP([]byte{1}, server.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	_, source, err := server.ReadFromUDP(make([]byte, 1))
	require.NoError(t, err)
	require.Equal(t, conn.localAddr.(*net.UDPAddr).Port, source.Port)
}
