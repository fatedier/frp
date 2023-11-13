// Copyright 2019 fatedier, fatedier@gmail.com
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
	"crypto/tls"
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/log"
	"net"
	"regexp"
	"time"

	libnet "github.com/fatedier/golib/net"
)

var FRPTLSHeadByte = 0x17

func CheckAndEnableTLSServerConnWithTimeout(
	c net.Conn, tlsConfig *tls.Config, tlsOnly bool, timeout time.Duration,
) (out net.Conn, isTLS bool, custom bool, err error) {
	sc, r := libnet.NewSharedConnSize(c, 2)
	buf := make([]byte, 1)
	var n int
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	n, err = r.Read(buf)
	_ = c.SetReadDeadline(time.Time{})
	if err != nil {
		return
	}

	switch {
	case n == 1 && int(buf[0]) == FRPTLSHeadByte:
		out = tls.Server(c, tlsConfig)
		isTLS = true
		custom = true
	case n == 1 && int(buf[0]) == 0x16:
		out = tls.Server(sc, tlsConfig)
		isTLS = true
	default:
		if tlsOnly {
			err = fmt.Errorf("non-TLS connection received on a TlsOnly server")
			return
		}
		out = sc
	}
	return
}

func IsClientCertificateSubjectValid(c net.Conn, tlsConfig v1.TLSServerConfig) bool {
	subjectRegex := tlsConfig.ClientCertificateSubjectRegex
	regex, err := regexp.Compile(subjectRegex)
	if err != nil {
		log.Trace("Client certificate subject validation is disabled")
		return true
	}

	tlsConn, ok := c.(*tls.Conn)
	if !ok {
		log.Warn("Skip client certificate subject validation because its not a tls connection")
		return true
	}

	state := tlsConn.ConnectionState()
	log.Trace("Validating client certificate subject using regex: %v", subjectRegex)
	if len(state.PeerCertificates) == 0 {
		log.Warn("No client certificates found in TLS connection, the verification was probably called to early.")
		return false
	}

	for _, v := range state.PeerCertificates {
		subject := fmt.Sprintf("%v", v.Subject)
		if !regex.MatchString(subject) {
			log.Warn("Client certificate subject %v doesn't match regex %v", v.Subject, subjectRegex)
			return false
		}
		log.Trace("Client certificate subject is valid")
	}
	return true
}
