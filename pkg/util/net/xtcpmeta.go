// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

var xtcpMetaMagic = [8]byte{'F', 'R', 'P', 'X', 'T', 'C', 'P', '1'}

func WriteXTCPClientMeta(w io.Writer, userConnRemote net.Addr) error {
	tcp, ok := userConnRemote.(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("unsupported addr type: %T", userConnRemote)
	}
	host := tcp.IP.String()
	if host == "" {
		host = "0.0.0.0"
	}
	if len(host) > 255 {
		return fmt.Errorf("src host too long")
	}
	buf := make([]byte, 0, 8+1+1+len(host)+2)
	buf = append(buf, xtcpMetaMagic[:]...)
	buf = append(buf, 0x01)
	buf = append(buf, byte(len(host)))
	buf = append(buf, []byte(host)...)
	p := make([]byte, 2)
	binary.BigEndian.PutUint16(p, uint16(tcp.Port))
	buf = append(buf, p...)
	_, err := w.Write(buf)
	return err
}

func ReadOrReplayXTCPClientMeta(rwc io.ReadWriteCloser) (newRWC io.ReadWriteCloser, srcHost string, srcPort uint16, ok bool, err error) {
	probe := make([]byte, 10)
	n, err := io.ReadFull(rwc, probe[:10])
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || err == io.EOF {
			return WrapPrependReadWriteCloser(rwc, probe[:n]), "", 0, false, nil
		}
		return WrapPrependReadWriteCloser(rwc, probe[:n]), "", 0, false, err
	}
	if string(probe[:8]) != string(xtcpMetaMagic[:]) || probe[8] != 0x01 {
		return WrapPrependReadWriteCloser(rwc, probe[:n]), "", 0, false, nil
	}
	hostLen := int(probe[9])
	need := 8 + 1 + 1 + hostLen + 2
	payload := make([]byte, need)
	copy(payload, probe[:10])
	if hostLen+2 > 0 {
		rest := payload[10:need]
		m, err2 := io.ReadFull(rwc, rest)
		if err2 != nil {
			// replay what we consumed so far to avoid corrupting stream
			return WrapPrependReadWriteCloser(rwc, payload[:10+m]), "", 0, false, nil
		}
	}
	host := string(payload[10 : 10+hostLen])
	port := binary.BigEndian.Uint16(payload[10+hostLen : need])
	return rwc, host, port, true, nil
}
