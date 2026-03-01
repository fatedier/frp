package source

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestBaseSourceLoadReturnsClonedConfigurers(t *testing.T) {
	require := require.New(t)

	src := NewConfigSource()

	proxyCfg := &v1.TCPProxyConfig{
		ProxyBaseConfig: v1.ProxyBaseConfig{
			Name: "proxy1",
			Type: "tcp",
		},
	}
	visitorCfg := &v1.STCPVisitorConfig{
		VisitorBaseConfig: v1.VisitorBaseConfig{
			Name: "visitor1",
			Type: "stcp",
		},
	}

	err := src.ReplaceAll([]v1.ProxyConfigurer{proxyCfg}, []v1.VisitorConfigurer{visitorCfg})
	require.NoError(err)

	firstProxies, firstVisitors, err := src.Load()
	require.NoError(err)
	require.Len(firstProxies, 1)
	require.Len(firstVisitors, 1)

	// Mutate loaded objects as runtime completion would do.
	firstProxies[0].Complete()
	firstVisitors[0].Complete()

	secondProxies, secondVisitors, err := src.Load()
	require.NoError(err)
	require.Len(secondProxies, 1)
	require.Len(secondVisitors, 1)

	require.Empty(secondProxies[0].GetBaseConfig().LocalIP)
	require.Empty(secondVisitors[0].GetBaseConfig().BindAddr)
}
