// Copyright 2025 Satyajeet Singh, jeet.0733@gmail.com
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
	"fmt"
	"net"
	"strings"

	libio "github.com/fatedier/golib/io"

	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
)

type HTTPSReverseProxy struct {
	vhostRouter *Routers
}

func NewHTTPSReverseProxy(vhostRouter *Routers) *HTTPSReverseProxy {
	return &HTTPSReverseProxy{
		vhostRouter: vhostRouter,
	}
}

// Register register the route config to reverse proxy
// reverse proxy will use CreateConnFn from routeCfg to create a connection to the remote service
func (rp *HTTPSReverseProxy) Register(routeCfg RouteConfig) error {
	err := rp.vhostRouter.Add(routeCfg.Domain, "", "", &routeCfg)
	if err != nil {
		return err
	}
	return nil
}

// UnRegister unregister route config by domain
func (rp *HTTPSReverseProxy) UnRegister(routeCfg RouteConfig) {
	rp.vhostRouter.Del(routeCfg.Domain, "", "")
}

func (rp *HTTPSReverseProxy) GetRouteConfig(domain string) *RouteConfig {
	// Validate and canonicalize hostname for security
	canonicalDomain, err := httppkg.CanonicalHost(domain)
	if err != nil {
		log.Debugf("invalid hostname [%s]: %v", domain, err)
		return nil
	}

	vr, ok := rp.getVhost(canonicalDomain)
	if ok {
		log.Debugf("get new https request host [%s]", canonicalDomain)
		return vr.payload.(*RouteConfig)
	}
	return nil
}

// CreateConnection create a new connection by route config
func (rp *HTTPSReverseProxy) CreateConnection(domain string) (net.Conn, error) {
	// Validate and canonicalize hostname for security
	canonicalDomain, err := httppkg.CanonicalHost(domain)
	if err != nil {
		return nil, fmt.Errorf("invalid hostname: %v", err)
	}

	vr, ok := rp.getVhost(canonicalDomain)
	if ok {
		fn := vr.payload.(*RouteConfig).CreateConnFn
		if fn != nil {
			return fn("")
		}
	}
	return nil, fmt.Errorf("%v: %s", ErrNoRouteFound, canonicalDomain)
}

// ProxyConn proxy connection for HTTPS
func (rp *HTTPSReverseProxy) ProxyConn(clientConn net.Conn, domain string) error {
	remoteConn, err := rp.CreateConnection(domain)
	if err != nil {
		return err
	}

	// Start proxying data between client and remote
	go func() {
		defer clientConn.Close()
		defer remoteConn.Close()
		libio.Join(clientConn, remoteConn)
	}()

	return nil
}

// getVhost tries to get vhost router by domain.
func (rp *HTTPSReverseProxy) getVhost(domain string) (*Router, bool) {
	findRouter := func(inDomain string) (*Router, bool) {
		vr, ok := rp.vhostRouter.Get(inDomain, "", "")
		if ok {
			return vr, ok
		}
		return nil, false
	}

	domain = strings.ToLower(domain)

	// Check the full hostname, if not exist, check the wildcard_domain such as *.example.com
	vr, ok := findRouter(domain)
	if ok {
		return vr, ok
	}

	// e.g. domain = test.example.com, try to match wildcard domains. *.example.com, *.com
	domainSplit := strings.Split(domain, ".")
	for len(domainSplit) >= 3 {
		domainSplit[0] = "*"
		wildcardDomain := strings.Join(domainSplit, ".")
		vr, ok = findRouter(wildcardDomain)
		if ok {
			return vr, true
		}
		domainSplit = domainSplit[1:]
	}

	// Finally, try to check if there is one proxy that domain is "*" means match all domains.
	vr, ok = findRouter("*")
	if ok {
		return vr, true
	}
	return nil, false
}
