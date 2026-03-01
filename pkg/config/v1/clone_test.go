package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProxyCloneDeepCopy(t *testing.T) {
	require := require.New(t)

	enabled := true
	pluginHTTP2 := true
	cfg := &HTTPProxyConfig{
		ProxyBaseConfig: ProxyBaseConfig{
			Name:        "p1",
			Type:        "http",
			Enabled:     &enabled,
			Annotations: map[string]string{"a": "1"},
			Metadatas:   map[string]string{"m": "1"},
			HealthCheck: HealthCheckConfig{
				Type: "http",
				HTTPHeaders: []HTTPHeader{
					{Name: "X-Test", Value: "v1"},
				},
			},
			ProxyBackend: ProxyBackend{
				Plugin: TypedClientPluginOptions{
					Type: PluginHTTPS2HTTP,
					ClientPluginOptions: &HTTPS2HTTPPluginOptions{
						Type:           PluginHTTPS2HTTP,
						EnableHTTP2:    &pluginHTTP2,
						RequestHeaders: HeaderOperations{Set: map[string]string{"k": "v"}},
					},
				},
			},
		},
		DomainConfig: DomainConfig{
			CustomDomains: []string{"a.example.com"},
			SubDomain:     "a",
		},
		Locations:       []string{"/api"},
		RequestHeaders:  HeaderOperations{Set: map[string]string{"h1": "v1"}},
		ResponseHeaders: HeaderOperations{Set: map[string]string{"h2": "v2"}},
	}

	cloned := cfg.Clone().(*HTTPProxyConfig)

	*cloned.Enabled = false
	cloned.Annotations["a"] = "changed"
	cloned.Metadatas["m"] = "changed"
	cloned.HealthCheck.HTTPHeaders[0].Value = "changed"
	cloned.CustomDomains[0] = "b.example.com"
	cloned.Locations[0] = "/new"
	cloned.RequestHeaders.Set["h1"] = "changed"
	cloned.ResponseHeaders.Set["h2"] = "changed"
	clientPlugin := cloned.Plugin.ClientPluginOptions.(*HTTPS2HTTPPluginOptions)
	*clientPlugin.EnableHTTP2 = false
	clientPlugin.RequestHeaders.Set["k"] = "changed"

	require.True(*cfg.Enabled)
	require.Equal("1", cfg.Annotations["a"])
	require.Equal("1", cfg.Metadatas["m"])
	require.Equal("v1", cfg.HealthCheck.HTTPHeaders[0].Value)
	require.Equal("a.example.com", cfg.CustomDomains[0])
	require.Equal("/api", cfg.Locations[0])
	require.Equal("v1", cfg.RequestHeaders.Set["h1"])
	require.Equal("v2", cfg.ResponseHeaders.Set["h2"])

	origPlugin := cfg.Plugin.ClientPluginOptions.(*HTTPS2HTTPPluginOptions)
	require.True(*origPlugin.EnableHTTP2)
	require.Equal("v", origPlugin.RequestHeaders.Set["k"])
}

func TestVisitorCloneDeepCopy(t *testing.T) {
	require := require.New(t)

	enabled := true
	cfg := &XTCPVisitorConfig{
		VisitorBaseConfig: VisitorBaseConfig{
			Name:       "v1",
			Type:       "xtcp",
			Enabled:    &enabled,
			ServerName: "server",
			BindPort:   7000,
			Plugin: TypedVisitorPluginOptions{
				Type: VisitorPluginVirtualNet,
				VisitorPluginOptions: &VirtualNetVisitorPluginOptions{
					Type:          VisitorPluginVirtualNet,
					DestinationIP: "10.0.0.1",
				},
			},
		},
		NatTraversal: &NatTraversalConfig{
			DisableAssistedAddrs: true,
		},
	}

	cloned := cfg.Clone().(*XTCPVisitorConfig)
	*cloned.Enabled = false
	cloned.NatTraversal.DisableAssistedAddrs = false
	visitorPlugin := cloned.Plugin.VisitorPluginOptions.(*VirtualNetVisitorPluginOptions)
	visitorPlugin.DestinationIP = "10.0.0.2"

	require.True(*cfg.Enabled)
	require.True(cfg.NatTraversal.DisableAssistedAddrs)
	origPlugin := cfg.Plugin.VisitorPluginOptions.(*VirtualNetVisitorPluginOptions)
	require.Equal("10.0.0.1", origPlugin.DestinationIP)
}
