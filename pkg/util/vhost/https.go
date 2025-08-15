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
	"crypto/tls"
	"io"
	"net"
	"strings"
	"time"

	"github.com/fatedier/golib/errors"
	libio "github.com/fatedier/golib/io"
	libnet "github.com/fatedier/golib/net"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type HTTPSMuxer struct {
	*Muxer
	httpsReverseProxy *HTTPSReverseProxy
}

func NewHTTPSMuxer(listener net.Listener, timeout time.Duration) (*HTTPSMuxer, error) {
	// Create muxer without auto-starting run method
	mux := &Muxer{
		listener:       listener,
		timeout:        timeout,
		vhostFunc:      GetHTTPSHostname,
		registryRouter: NewRouters(),
	}
	mux.SetFailHookFunc(vhostFailed)

	httpsMux := &HTTPSMuxer{Muxer: mux}
	// Start our custom handler
	go httpsMux.run()
	return httpsMux, nil
}

func (h *HTTPSMuxer) SetHTTPSReverseProxy(rp *HTTPSReverseProxy) {
	h.httpsReverseProxy = rp
}

// Override the base muxer's run method to handle group routing
func (h *HTTPSMuxer) run() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			return
		}
		go h.handleHTTPS(conn)
	}
}

func (h *HTTPSMuxer) handleHTTPS(c net.Conn) {
	if err := c.SetDeadline(time.Now().Add(h.timeout)); err != nil {
		_ = c.Close()
		return
	}

	sConn, reqInfoMap, err := h.vhostFunc(c)
	if err != nil {
		log.Debugf("get hostname from https request error: %v", err)
		_ = c.Close()
		return
	}

	hostname := strings.ToLower(reqInfoMap["Host"])

	// Validate and canonicalize hostname for security
	canonicalHostname, err := httppkg.CanonicalHost(hostname)
	if err != nil {
		log.Debugf("invalid hostname [%s]: %v", hostname, err)
		h.failHook(sConn)
		return
	}

	// First check if there's a group route for this domain
	if h.httpsReverseProxy != nil {
		if routeConfig := h.httpsReverseProxy.GetRouteConfig(canonicalHostname); routeConfig != nil {
			log.Debugf("routing https request for host [%s] to group", hostname)

			// SECURITY: Apply authentication check before group routing
			if routeConfig.Username != "" && routeConfig.Password != "" {
				if h.checkAuth != nil {
					ok, err := h.checkAuth(c, routeConfig.Username, routeConfig.Password, reqInfoMap)
					if !ok || err != nil {
						log.Debugf("auth failed for group route user: %s", routeConfig.Username)
						h.failHook(sConn)
						return
					}
				}
			}

			// SECURITY: Apply success hook for group routing
			if h.successHook != nil {
				if err := h.successHook(c, reqInfoMap); err != nil {
					log.Infof("success func failure on group vhost connection: %v", err)
					_ = c.Close()
					return
				}
			}

			if err = sConn.SetDeadline(time.Time{}); err != nil {
				_ = c.Close()
				return
			}

			// Create connection to backend through group routing
			remoteConn, err := h.httpsReverseProxy.CreateConnection(canonicalHostname)
			if err != nil {
				log.Debugf("failed to create connection through group: %v", err)
				h.failHook(sConn)
				return
			}

			// Start proxying data between client and remote
			go func() {
				defer func() {
					if err := recover(); err != nil {
						log.Warnf("panic in HTTPS proxy goroutine: %v", err)
					}
				}()
				defer sConn.Close()
				defer remoteConn.Close()
				libio.Join(sConn, remoteConn)
			}()
			return
		}
	}

	// Fall back to direct listener routing (existing behavior)
	path := strings.ToLower(reqInfoMap["Path"])
	httpUser := reqInfoMap["HTTPUser"]
	l, ok := h.getListener(canonicalHostname, path, httpUser)
	if !ok {
		log.Debugf("https request for host [%s] path [%s] httpUser [%s] not found", hostname, path, httpUser)
		h.failHook(sConn)
		return
	}

	xl := xlog.FromContextSafe(l.ctx)
	if h.successHook != nil {
		if err := h.successHook(c, reqInfoMap); err != nil {
			xl.Infof("success func failure on vhost connection: %v", err)
			_ = c.Close()
			return
		}
	}

	// if checkAuth func is exist and username/password is set
	// then verify user access
	if l.mux.checkAuth != nil && l.username != "" {
		ok, err := l.mux.checkAuth(c, l.username, l.password, reqInfoMap)
		if !ok || err != nil {
			xl.Debugf("auth failed for user: %s", l.username)
			_ = c.Close()
			return
		}
	}

	if err = sConn.SetDeadline(time.Time{}); err != nil {
		_ = c.Close()
		return
	}
	c = sConn

	xl.Debugf("new https request host [%s] path [%s] httpUser [%s]", hostname, path, httpUser)
	err = errors.PanicToError(func() {
		l.accept <- c
	})
	if err != nil {
		xl.Warnf("listener is already closed, ignore this request")
	}
}

func GetHTTPSHostname(c net.Conn) (_ net.Conn, _ map[string]string, err error) {
	reqInfoMap := make(map[string]string, 0)
	sc, rd := libnet.NewSharedConn(c)

	clientHello, err := readClientHello(rd)
	if err != nil {
		return nil, reqInfoMap, err
	}

	reqInfoMap["Host"] = clientHello.ServerName
	reqInfoMap["Scheme"] = "https"
	return sc, reqInfoMap, nil
}

func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo

	// Note that Handshake always fails because the readOnlyConn is not a real connection.
	// As long as the Client Hello is successfully read, the failure should only happen after GetConfigForClient is called,
	// so we only care about the error if hello was never set.
	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = &tls.ClientHelloInfo{}
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()

	if hello == nil {
		return nil, err
	}
	return hello, nil
}

func vhostFailed(c net.Conn) {
	// Alert with alertUnrecognizedName
	_ = tls.Server(c, &tls.Config{}).Handshake()
	c.Close()
}

type readOnlyConn struct {
	reader io.Reader
}

func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(_ []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(_ time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(_ time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(_ time.Time) error { return nil }
