// +build !linux

package kcp

import (
	"golang.org/x/net/ipv4"
)

func (s *UDPSession) tx(txqueue []ipv4.Message) {
	s.defaultTx(txqueue)
}
