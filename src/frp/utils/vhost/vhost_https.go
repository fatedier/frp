// Copyright 2016 fatedier, fatedier@gmail.com
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

package vhost

import (
	_ "bufio"
	_ "bytes"
	_ "crypto/tls"
	"errors"
	"fmt"
	"frp/utils/conn"
	"frp/utils/log"
	"io"
	_ "io/ioutil"
	"net"
	_ "net/http"
	"strings"
	_ "sync"
	"time"
)

var (
	maxHandshake   int64 = 65536 // maximum handshake we support (protocol max is 16 MB)
	VhostHttpsPort int64 = 443
)

const (
	typeClientHello uint8 = 1 // Type client hello
)

// TLS extension numbers
const (
	extensionServerName          uint16 = 0
	extensionStatusRequest       uint16 = 5
	extensionSupportedCurves     uint16 = 10
	extensionSupportedPoints     uint16 = 11
	extensionSignatureAlgorithms uint16 = 13
	extensionALPN                uint16 = 16
	extensionSCT                 uint16 = 18
	extensionSessionTicket       uint16 = 35
	extensionNextProtoNeg        uint16 = 13172 // not IANA assigned
	extensionRenegotiationInfo   uint16 = 0xff01
)

type HttpsMuxer struct {
	*VhostMuxer
}

/*
   RFC document: http://tools.ietf.org/html/rfc5246
*/

func errMsgToLog(format string, a ...interface{}) error {
	errMsg := fmt.Sprintf(format, a...)
	log.Warn(errMsg)
	return errors.New(errMsg)
}

func readHandshake(rd io.Reader) (string, error) {

	data := make([]byte, 1024)
	length, err := rd.Read(data)
	if err != nil {
		return "", errMsgToLog("read err:%v", err)
	} else {
		if length < 47 {
			return "", errMsgToLog("readHandshake: proto length[%d] is too short", length)
		}
	}
	data = data[:length]
	//log.Warn("data: %+v", data)
	if uint8(data[5]) != typeClientHello {
		return "", errMsgToLog("readHandshake: type[%d] is not clientHello", uint16(data[5]))
	}

	//version and random
	//tlsVersion := uint16(data[9])<<8 | uint16(data[10])
	//random := data[11:43]

	//session
	sessionIdLen := int(data[43])
	if sessionIdLen > 32 || len(data) < 44+sessionIdLen {
		return "", errMsgToLog("readHandshake: sessionIdLen[%d] is long", sessionIdLen)
	}
	data = data[44+sessionIdLen:]
	if len(data) < 2 {
		return "", errMsgToLog("readHandshake: dataLen[%d] after session is short", len(data))
	}

	// cipher suite numbers
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		//return "", errMsgToLog("readHandshake: cipherSuiteLen[%d] is long", sessionIdLen)
		return "", errMsgToLog("readHandshake: dataLen[%d] after cipher suite is short", len(data))
	}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		return "", errMsgToLog("readHandshake: cipherSuiteLen[%d] is long", cipherSuiteLen)
	}

	//compression method
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return "", errMsgToLog("readHandshake: compressionMethodsLen[%d] is long", compressionMethodsLen)
		//return false
	}

	data = data[1+compressionMethodsLen:]

	if len(data) == 0 {
		// ClientHello is optionally followed by extension data
		//return true
		return "", errMsgToLog("readHandshake: there is no extension data to get servername")
	}
	if len(data) < 2 {
		return "", errMsgToLog("readHandshake: extension dataLen[%d] is too short")
	}

	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if extensionsLength != len(data) {
		return "", errMsgToLog("readHandshake: extensionsLen[%d] is not equal to dataLen[%d]", extensionsLength, len(data))
	}
	for len(data) != 0 {
		if len(data) < 4 {
			return "", errMsgToLog("readHandshake: extensionsDataLen[%d] is too short", len(data))
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < length {
			return "", errMsgToLog("readHandshake: extensionLen[%d] is long", length)
			//return false
		}

		switch extension {
		case extensionRenegotiationInfo:
			if length != 1 || data[0] != 0 {
				return "", errMsgToLog("readHandshake: extension reNegotiationInfoLen[%d] is short", length)
			}
		case extensionNextProtoNeg:
		case extensionStatusRequest:
		case extensionServerName:
			d := data[:length]
			if len(d) < 2 {
				return "", errMsgToLog("readHandshake: remiaining dataLen[%d] is short", len(d))
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				return "", errMsgToLog("readHandshake: nameListLen[%d] is not equal to dataLen[%d]", namesLen, len(d))
			}
			for len(d) > 0 {
				if len(d) < 3 {
					return "", errMsgToLog("readHandshake: extension serverNameLen[%d] is short", len(d))
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					return "", errMsgToLog("readHandshake: nameLen[%d] is not equal to dataLen[%d]", nameLen, len(d))
				}
				if nameType == 0 {
					suffix := ""
					if VhostHttpsPort != 443 {
						suffix = fmt.Sprintf(":%d", VhostHttpsPort)
					}
					serverName := string(d[:nameLen])
					domain := strings.ToLower(strings.TrimSpace(serverName)) + suffix
					return domain, nil
					break
				}
				d = d[nameLen:]
			}
		}
		data = data[length:]
	}
	//return "test.codermao.com:8082", nil
	return "", errMsgToLog("Unknow error")
}

func GetHttpsHostname(c *conn.Conn) (sc net.Conn, routerName string, err error) {
	log.Info("GetHttpsHostname")
	sc, rd := newShareConn(c.TcpConn)

	host, err := readHandshake(rd)
	if err != nil {
		return sc, "", err
	}
	/*
		if _, ok := c.TcpConn.(*tls.Conn); ok {
			log.Warn("convert to tlsConn success")
		} else {
			log.Warn("convert to tlsConn error")
		}*/
	//tcpConn.
	log.Debug("GetHttpsHostname: %s", host)

	return sc, host, nil
}

func NewHttpsMuxer(listener *conn.Listener, timeout time.Duration) (*HttpsMuxer, error) {
	mux, err := NewVhostMuxer(listener, GetHttpsHostname, timeout)
	return &HttpsMuxer{mux}, err
}
