package group

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/util/tcpmux"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func TestHTTPGroupMultipleRoutes(t *testing.T) {
	router := vhost.NewRouters()
	rp := vhost.NewHTTPReverseProxy(vhost.HTTPReverseProxyOptions{}, router)
	ctl := NewHTTPGroupController(router)

	newRoute := func(proxyName, domain, location string) vhost.RouteConfig {
		return vhost.RouteConfig{
			Domain:   domain,
			Location: location,
			CreateConnFn: func(string) (net.Conn, error) {
				return nil, &net.AddrError{Err: proxyName}
			},
		}
	}
	domains := []string{"a.example.com", "b.example.com"}
	locations := []string{"/x", "/y"}

	// One proxy with several domains and locations registers one route per
	// domain/location pair, and every member of the group serves all of them.
	for _, proxyName := range []string{"foo", "bar"} {
		for _, domain := range domains {
			for _, location := range locations {
				require.NoError(t, ctl.Register(proxyName, "web", "123", newRoute(proxyName, domain, location)))
			}
		}
	}

	served := func(domain, location string) map[string]bool {
		rc := rp.GetRouteConfig(domain, location, "")
		require.NotNil(t, rc, "%s%s is not routed", domain, location)
		names := make(map[string]bool)
		for range 4 {
			_, err := rc.CreateConnFn("")
			names[err.(*net.AddrError).Err] = true
		}
		return names
	}
	for _, domain := range domains {
		for _, location := range locations {
			assert.Equal(t, map[string]bool{"foo": true, "bar": true}, served(domain, location), "%s%s", domain, location)
		}
	}

	assert.ErrorIs(t, ctl.Register("baz", "web", "456", newRoute("baz", "a.example.com", "/x")), ErrGroupAuthFailed)
	assert.ErrorIs(t, ctl.Register("foo", "web", "123", newRoute("foo", "a.example.com", "/x")), ErrProxyRepeated)

	// Removing a proxy keeps the other member on every route, and removing
	// the last member releases the routes.
	for _, domain := range domains {
		for _, location := range locations {
			ctl.UnRegister("foo", "web", newRoute("foo", domain, location))
		}
	}
	for _, domain := range domains {
		for _, location := range locations {
			assert.Equal(t, map[string]bool{"bar": true}, served(domain, location), "%s%s", domain, location)
		}
	}
	for _, domain := range domains {
		for _, location := range locations {
			ctl.UnRegister("bar", "web", newRoute("bar", domain, location))
			assert.Nil(t, rp.GetRouteConfig(domain, location, ""), "%s%s is still routed", domain, location)
		}
	}
	assert.Empty(t, ctl.groups)
}

func TestHTTPSAndTCPMuxGroupMultipleDomains(t *testing.T) {
	newListener := func() net.Listener {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { l.Close() })
		return l
	}
	httpsMuxer, err := vhost.NewHTTPSMuxer(newListener(), time.Second)
	require.NoError(t, err)
	httpsCtl := NewHTTPSGroupController(httpsMuxer)

	tcpMuxer, err := tcpmux.NewHTTPConnectTCPMuxer(newListener(), false, time.Second)
	require.NoError(t, err)
	tcpMuxCtl := NewTCPMuxGroupCtl(tcpMuxer)

	listen := map[string]func(domain string) (net.Listener, error){
		"https": func(domain string) (net.Listener, error) {
			return httpsCtl.Listen(context.Background(), "web", "123", vhost.RouteConfig{Domain: domain})
		},
		"tcpmux": func(domain string) (net.Listener, error) {
			return tcpMuxCtl.Listen(context.Background(), "httpconnect", "web", "123", vhost.RouteConfig{Domain: domain})
		},
	}
	for name, fn := range listen {
		t.Run(name, func(t *testing.T) {
			lns := make([]net.Listener, 0, 4)
			// Two proxies of the same group, each with two domains.
			for range 2 {
				for _, domain := range []string{"a.example.com", "b.example.com"} {
					ln, err := fn(domain)
					require.NoError(t, err, domain)
					lns = append(lns, ln)
				}
			}
			for _, ln := range lns {
				require.NoError(t, ln.Close())
			}
		})
	}
	assert.Empty(t, httpsCtl.groups)
	assert.Empty(t, tcpMuxCtl.groups)
}
