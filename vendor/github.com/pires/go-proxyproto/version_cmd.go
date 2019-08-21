package proxyproto

// ProtocolVersionAndCommand represents proxy protocol version and command.
type ProtocolVersionAndCommand byte

const (
	LOCAL = '\x20'
	PROXY = '\x21'
)

var supportedCommand = map[ProtocolVersionAndCommand]bool{
	LOCAL: true,
	PROXY: true,
}

// IsLocal returns true if the protocol version is \x2 and command is LOCAL, false otherwise.
func (pvc ProtocolVersionAndCommand) IsLocal() bool {
	return 0x20 == pvc&0xF0 && 0x00 == pvc&0x0F
}

// IsProxy returns true if the protocol version is \x2 and command is PROXY, false otherwise.
func (pvc ProtocolVersionAndCommand) IsProxy() bool {
	return 0x20 == pvc&0xF0 && 0x01 == pvc&0x0F
}

// IsUnspec returns true if the protocol version or command is unspecified, false otherwise.
func (pvc ProtocolVersionAndCommand) IsUnspec() bool {
	return !(pvc.IsLocal() || pvc.IsProxy())
}

func (pvc ProtocolVersionAndCommand) toByte() byte {
	if pvc.IsLocal() {
		return LOCAL
	} else if pvc.IsProxy() {
		return PROXY
	}

	return LOCAL
}
