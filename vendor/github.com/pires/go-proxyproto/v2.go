package proxyproto

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

var (
	lengthV4   = uint16(12)
	lengthV6   = uint16(36)
	lengthUnix = uint16(218)

	lengthV4Bytes = func() []byte {
		a := make([]byte, 2)
		binary.BigEndian.PutUint16(a, lengthV4)
		return a
	}()
	lengthV6Bytes = func() []byte {
		a := make([]byte, 2)
		binary.BigEndian.PutUint16(a, lengthV6)
		return a
	}()
	lengthUnixBytes = func() []byte {
		a := make([]byte, 2)
		binary.BigEndian.PutUint16(a, lengthUnix)
		return a
	}()
)

type _ports struct {
	SrcPort uint16
	DstPort uint16
}

type _addr4 struct {
	Src     [4]byte
	Dst     [4]byte
	SrcPort uint16
	DstPort uint16
}

type _addr6 struct {
	Src [16]byte
	Dst [16]byte
	_ports
}

type _addrUnix struct {
	Src [108]byte
	Dst [108]byte
}

func parseVersion2(reader *bufio.Reader) (header *Header, err error) {
	// Skip first 12 bytes (signature)
	for i := 0; i < 12; i++ {
		if _, err = reader.ReadByte(); err != nil {
			return nil, ErrCantReadProtocolVersionAndCommand
		}
	}

	header = new(Header)
	header.Version = 2

	// Read the 13th byte, protocol version and command
	b13, err := reader.ReadByte()
	if err != nil {
		return nil, ErrCantReadProtocolVersionAndCommand
	}
	header.Command = ProtocolVersionAndCommand(b13)
	if _, ok := supportedCommand[header.Command]; !ok {
		return nil, ErrUnsupportedProtocolVersionAndCommand
	}
	// If command is LOCAL, header ends here
	if header.Command.IsLocal() {
		return header, nil
	}

	// Read the 14th byte, address family and protocol
	b14, err := reader.ReadByte()
	if err != nil {
		return nil, ErrCantReadAddressFamilyAndProtocol
	}
	header.TransportProtocol = AddressFamilyAndProtocol(b14)
	if _, ok := supportedTransportProtocol[header.TransportProtocol]; !ok {
		return nil, ErrUnsupportedAddressFamilyAndProtocol
	}

	// Make sure there are bytes available as specified in length
	var length uint16
	if err := binary.Read(io.LimitReader(reader, 2), binary.BigEndian, &length); err != nil {
		return nil, ErrCantReadLength
	}
	if !header.validateLength(length) {
		return nil, ErrInvalidLength
	}

	if _, err := reader.Peek(int(length)); err != nil {
		return nil, ErrInvalidLength
	}

	// Length-limited reader for payload section
	payloadReader := io.LimitReader(reader, int64(length))

	// Read addresses and ports
	if header.TransportProtocol.IsIPv4() {
		var addr _addr4
		if err := binary.Read(payloadReader, binary.BigEndian, &addr); err != nil {
			return nil, ErrInvalidAddress
		}
		header.SourceAddress = addr.Src[:]
		header.DestinationAddress = addr.Dst[:]
		header.SourcePort = addr.SrcPort
		header.DestinationPort = addr.DstPort
	} else if header.TransportProtocol.IsIPv6() {
		var addr _addr6
		if err := binary.Read(payloadReader, binary.BigEndian, &addr); err != nil {
			return nil, ErrInvalidAddress
		}
		header.SourceAddress = addr.Src[:]
		header.DestinationAddress = addr.Dst[:]
		header.SourcePort = addr.SrcPort
		header.DestinationPort = addr.DstPort
	}
	// TODO fully support Unix addresses
	//	else if header.TransportProtocol.IsUnix() {
	//		var addr _addrUnix
	//		if err := binary.Read(payloadReader, binary.BigEndian, &addr); err != nil {
	//			return nil, ErrInvalidAddress
	//		}
	//
	//if header.SourceAddress, err = net.ResolveUnixAddr("unix", string(addr.Src[:])); err != nil {
	//	return nil, ErrCantResolveSourceUnixAddress
	//}
	//if header.DestinationAddress, err = net.ResolveUnixAddr("unix", string(addr.Dst[:])); err != nil {
	//	return nil, ErrCantResolveDestinationUnixAddress
	//}
	//}

	// TODO add encapsulated TLV support

	// Drain the remaining padding
	payloadReader.Read(make([]byte, length))

	return header, nil
}

func (header *Header) writeVersion2(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.Write(SIGV2)
	buf.WriteByte(header.Command.toByte())
	if !header.Command.IsLocal() {
		buf.WriteByte(header.TransportProtocol.toByte())
		// TODO add encapsulated TLV length
		var addrSrc, addrDst []byte
		if header.TransportProtocol.IsIPv4() {
			buf.Write(lengthV4Bytes)
			addrSrc = header.SourceAddress.To4()
			addrDst = header.DestinationAddress.To4()
		} else if header.TransportProtocol.IsIPv6() {
			buf.Write(lengthV6Bytes)
			addrSrc = header.SourceAddress.To16()
			addrDst = header.DestinationAddress.To16()
		} else if header.TransportProtocol.IsUnix() {
			buf.Write(lengthUnixBytes)
			// TODO is below right?
			addrSrc = []byte(header.SourceAddress.String())
			addrDst = []byte(header.DestinationAddress.String())
		}
		buf.Write(addrSrc)
		buf.Write(addrDst)

		portSrcBytes := func() []byte {
			a := make([]byte, 2)
			binary.BigEndian.PutUint16(a, header.SourcePort)
			return a
		}()
		buf.Write(portSrcBytes)

		portDstBytes := func() []byte {
			a := make([]byte, 2)
			binary.BigEndian.PutUint16(a, header.DestinationPort)
			return a
		}()
		buf.Write(portDstBytes)

	}

	return buf.WriteTo(w)
}

func (header *Header) validateLength(length uint16) bool {
	if header.TransportProtocol.IsIPv4() {
		return length >= lengthV4
	} else if header.TransportProtocol.IsIPv6() {
		return length >= lengthV6
	} else if header.TransportProtocol.IsUnix() {
		return length >= lengthUnix
	}
	return false
}
