package proxy

import (
	"reflect"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func init() {
	pxyConfs := []v1.ProxyConfigurer{
		&v1.TCPProxyConfig{},
		&v1.HTTPProxyConfig{},
		&v1.HTTPSProxyConfig{},
		&v1.STCPProxyConfig{},
		&v1.TCPMuxProxyConfig{},
		&v1.STCPProxyConfig{},
	}
	for _, cfg := range pxyConfs {
		RegisterProxyFactory(reflect.TypeOf(cfg), NewGeneralTCPProxy)
	}
}

// GeneralTCPProxy is a general implementation of Proxy interface for TCP protocol.
// If the default GeneralTCPProxy cannot meet the requirements, you can customize
// the implementation of the Proxy interface.
type GeneralTCPProxy struct {
	*BaseProxy
}

func NewGeneralTCPProxy(baseProxy *BaseProxy, _ v1.ProxyConfigurer) Proxy {
	return &GeneralTCPProxy{
		BaseProxy: baseProxy,
	}
}
