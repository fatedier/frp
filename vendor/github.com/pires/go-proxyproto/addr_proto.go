package proxyproto

// AddressFamilyAndProtocol represents address family and transport protocol.
type AddressFamilyAndProtocol byte

const (
	UNSPEC       = '\x00'
	TCPv4        = '\x11'
	UDPv4        = '\x12'
	TCPv6        = '\x21'
	UDPv6        = '\x22'
	UnixStream   = '\x31'
	UnixDatagram = '\x32'
)

var supportedTransportProtocol = map[AddressFamilyAndProtocol]bool{
	TCPv4:        true,
	UDPv4:        true,
	TCPv6:        true,
	UDPv6:        true,
	UnixStream:   true,
	UnixDatagram: true,
}

// IsIPv4 returns true if the address family is IPv4 (AF_INET4), false otherwise.
func (ap AddressFamilyAndProtocol) IsIPv4() bool {
	return 0x10 == ap&0xF0
}

// IsIPv6 returns true if the address family is IPv6 (AF_INET6), false otherwise.
func (ap AddressFamilyAndProtocol) IsIPv6() bool {
	return 0x20 == ap&0xF0
}

// IsUnix returns true if the address family is UNIX (AF_UNIX), false otherwise.
func (ap AddressFamilyAndProtocol) IsUnix() bool {
	return 0x30 == ap&0xF0
}

// IsStream returns true if the transport protocol is TCP or STREAM (SOCK_STREAM), false otherwise.
func (ap AddressFamilyAndProtocol) IsStream() bool {
	return 0x01 == ap&0x0F
}

// IsDatagram returns true if the transport protocol is UDP or DGRAM (SOCK_DGRAM), false otherwise.
func (ap AddressFamilyAndProtocol) IsDatagram() bool {
	return 0x02 == ap&0x0F
}

// IsUnspec returns true if the transport protocol or address family is unspecified, false otherwise.
func (ap AddressFamilyAndProtocol) IsUnspec() bool {
	return (0x00 == ap&0xF0) || (0x00 == ap&0x0F)
}

func (ap AddressFamilyAndProtocol) toByte() byte {
	if ap.IsIPv4() && ap.IsStream() {
		return TCPv4
	} else if ap.IsIPv4() && ap.IsDatagram() {
		return UDPv4
	} else if ap.IsIPv6() && ap.IsStream() {
		return TCPv6
	} else if ap.IsIPv6() && ap.IsDatagram() {
		return UDPv6
	} else if ap.IsUnix() && ap.IsStream() {
		return UnixStream
	} else if ap.IsUnix() && ap.IsDatagram() {
		return UnixDatagram
	}

	return UNSPEC
}
