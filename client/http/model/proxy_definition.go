package model

import (
	"fmt"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type ProxyDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`

	TCP    *v1.TCPProxyConfig    `json:"tcp,omitempty"`
	UDP    *v1.UDPProxyConfig    `json:"udp,omitempty"`
	HTTP   *v1.HTTPProxyConfig   `json:"http,omitempty"`
	HTTPS  *v1.HTTPSProxyConfig  `json:"https,omitempty"`
	TCPMux *v1.TCPMuxProxyConfig `json:"tcpmux,omitempty"`
	STCP   *v1.STCPProxyConfig   `json:"stcp,omitempty"`
	SUDP   *v1.SUDPProxyConfig   `json:"sudp,omitempty"`
	XTCP   *v1.XTCPProxyConfig   `json:"xtcp,omitempty"`
}

func (p *ProxyDefinition) Validate(pathName string, isUpdate bool) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("proxy name is required")
	}
	if !IsProxyType(p.Type) {
		return fmt.Errorf("invalid proxy type: %s", p.Type)
	}
	if isUpdate && pathName != "" && pathName != p.Name {
		return fmt.Errorf("proxy name in URL must match name in body")
	}

	_, blockType, blockCount := p.activeBlock()
	if blockCount != 1 {
		return fmt.Errorf("exactly one proxy type block is required")
	}
	if blockType != p.Type {
		return fmt.Errorf("proxy type block %q does not match type %q", blockType, p.Type)
	}
	return nil
}

func (p *ProxyDefinition) ToConfigurer() (v1.ProxyConfigurer, error) {
	block, _, _ := p.activeBlock()
	if block == nil {
		return nil, fmt.Errorf("exactly one proxy type block is required")
	}

	cfg := block
	cfg.GetBaseConfig().Name = p.Name
	cfg.GetBaseConfig().Type = p.Type
	return cfg, nil
}

func ProxyDefinitionFromConfigurer(cfg v1.ProxyConfigurer) (ProxyDefinition, error) {
	if cfg == nil {
		return ProxyDefinition{}, fmt.Errorf("proxy config is nil")
	}

	base := cfg.GetBaseConfig()
	payload := ProxyDefinition{
		Name: base.Name,
		Type: base.Type,
	}

	switch c := cfg.(type) {
	case *v1.TCPProxyConfig:
		payload.TCP = c
	case *v1.UDPProxyConfig:
		payload.UDP = c
	case *v1.HTTPProxyConfig:
		payload.HTTP = c
	case *v1.HTTPSProxyConfig:
		payload.HTTPS = c
	case *v1.TCPMuxProxyConfig:
		payload.TCPMux = c
	case *v1.STCPProxyConfig:
		payload.STCP = c
	case *v1.SUDPProxyConfig:
		payload.SUDP = c
	case *v1.XTCPProxyConfig:
		payload.XTCP = c
	default:
		return ProxyDefinition{}, fmt.Errorf("unsupported proxy configurer type %T", cfg)
	}

	return payload, nil
}

func (p *ProxyDefinition) activeBlock() (v1.ProxyConfigurer, string, int) {
	count := 0
	var block v1.ProxyConfigurer
	var blockType string

	if p.TCP != nil {
		count++
		block = p.TCP
		blockType = "tcp"
	}
	if p.UDP != nil {
		count++
		block = p.UDP
		blockType = "udp"
	}
	if p.HTTP != nil {
		count++
		block = p.HTTP
		blockType = "http"
	}
	if p.HTTPS != nil {
		count++
		block = p.HTTPS
		blockType = "https"
	}
	if p.TCPMux != nil {
		count++
		block = p.TCPMux
		blockType = "tcpmux"
	}
	if p.STCP != nil {
		count++
		block = p.STCP
		blockType = "stcp"
	}
	if p.SUDP != nil {
		count++
		block = p.SUDP
		blockType = "sudp"
	}
	if p.XTCP != nil {
		count++
		block = p.XTCP
		blockType = "xtcp"
	}

	return block, blockType, count
}

func IsProxyType(typ string) bool {
	switch typ {
	case "tcp", "udp", "http", "https", "tcpmux", "stcp", "sudp", "xtcp":
		return true
	default:
		return false
	}
}
