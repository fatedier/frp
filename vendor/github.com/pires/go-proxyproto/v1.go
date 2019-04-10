package proxyproto

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strconv"
	"strings"
)

const (
	CRLF      = "\r\n"
	SEPARATOR = " "
)

func initVersion1() *Header {
	header := new(Header)
	header.Version = 1
	// Command doesn't exist in v1
	header.Command = PROXY
	return header
}

func parseVersion1(reader *bufio.Reader) (*Header, error) {
	// Make sure we have a v1 header
	line, err := reader.ReadString('\n')
	if !strings.HasSuffix(line, CRLF) {
		return nil, ErrCantReadProtocolVersionAndCommand
	}
	tokens := strings.Split(line[:len(line)-2], SEPARATOR)
	if len(tokens) < 6 {
		return nil, ErrCantReadProtocolVersionAndCommand
	}

	header := initVersion1()

	// Read address family and protocol
	switch tokens[1] {
	case "TCP4":
		header.TransportProtocol = TCPv4
	case "TCP6":
		header.TransportProtocol = TCPv6
	default:
		header.TransportProtocol = UNSPEC
	}

	// Read addresses and ports
	header.SourceAddress, err = parseV1IPAddress(header.TransportProtocol, tokens[2])
	if err != nil {
		return nil, err
	}
	header.DestinationAddress, err = parseV1IPAddress(header.TransportProtocol, tokens[3])
	if err != nil {
		return nil, err
	}
	header.SourcePort, err = parseV1PortNumber(tokens[4])
	if err != nil {
		return nil, err
	}
	header.DestinationPort, err = parseV1PortNumber(tokens[5])
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (header *Header) writeVersion1(w io.Writer) (int64, error) {
	// As of version 1, only "TCP4" ( \x54 \x43 \x50 \x34 ) for TCP over IPv4,
	// and "TCP6" ( \x54 \x43 \x50 \x36 ) for TCP over IPv6 are allowed.
	proto := "UNKNOWN"
	if header.TransportProtocol == TCPv4 {
		proto = "TCP4"
	} else if header.TransportProtocol == TCPv6 {
		proto = "TCP6"
	}

	var buf bytes.Buffer
	buf.Write(SIGV1)
	buf.WriteString(SEPARATOR)
	buf.WriteString(proto)
	buf.WriteString(SEPARATOR)
	buf.WriteString(header.SourceAddress.String())
	buf.WriteString(SEPARATOR)
	buf.WriteString(header.DestinationAddress.String())
	buf.WriteString(SEPARATOR)
	buf.WriteString(strconv.Itoa(int(header.SourcePort)))
	buf.WriteString(SEPARATOR)
	buf.WriteString(strconv.Itoa(int(header.DestinationPort)))
	buf.WriteString(CRLF)

	return buf.WriteTo(w)
}

func parseV1PortNumber(portStr string) (uint16, error) {
	var port uint16

	_port, err := strconv.Atoi(portStr)
	if err == nil {
		if port < 0 || port > 65535 {
			err = ErrInvalidPortNumber
		}
		port = uint16(_port)
	}

	return port, err
}

func parseV1IPAddress(protocol AddressFamilyAndProtocol, addrStr string) (addr net.IP, err error) {
	addr = net.ParseIP(addrStr)
	tryV4 := addr.To4()
	if (protocol == TCPv4 && tryV4 == nil) || (protocol == TCPv6 && tryV4 != nil) {
		err = ErrInvalidAddress
	}
	return
}
