package model

import (
	"fmt"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type VisitorDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`

	STCP *v1.STCPVisitorConfig `json:"stcp,omitempty"`
	SUDP *v1.SUDPVisitorConfig `json:"sudp,omitempty"`
	XTCP *v1.XTCPVisitorConfig `json:"xtcp,omitempty"`
}

func (p *VisitorDefinition) Validate(pathName string, isUpdate bool) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("visitor name is required")
	}
	if !IsVisitorType(p.Type) {
		return fmt.Errorf("invalid visitor type: %s", p.Type)
	}
	if isUpdate && pathName != "" && pathName != p.Name {
		return fmt.Errorf("visitor name in URL must match name in body")
	}

	_, blockType, blockCount := p.activeBlock()
	if blockCount != 1 {
		return fmt.Errorf("exactly one visitor type block is required")
	}
	if blockType != p.Type {
		return fmt.Errorf("visitor type block %q does not match type %q", blockType, p.Type)
	}
	return nil
}

func (p *VisitorDefinition) ToConfigurer() (v1.VisitorConfigurer, error) {
	block, _, _ := p.activeBlock()
	if block == nil {
		return nil, fmt.Errorf("exactly one visitor type block is required")
	}

	cfg := block
	cfg.GetBaseConfig().Name = p.Name
	cfg.GetBaseConfig().Type = p.Type
	return cfg, nil
}

func VisitorDefinitionFromConfigurer(cfg v1.VisitorConfigurer) (VisitorDefinition, error) {
	if cfg == nil {
		return VisitorDefinition{}, fmt.Errorf("visitor config is nil")
	}

	base := cfg.GetBaseConfig()
	payload := VisitorDefinition{
		Name: base.Name,
		Type: base.Type,
	}

	switch c := cfg.(type) {
	case *v1.STCPVisitorConfig:
		payload.STCP = c
	case *v1.SUDPVisitorConfig:
		payload.SUDP = c
	case *v1.XTCPVisitorConfig:
		payload.XTCP = c
	default:
		return VisitorDefinition{}, fmt.Errorf("unsupported visitor configurer type %T", cfg)
	}

	return payload, nil
}

func (p *VisitorDefinition) activeBlock() (v1.VisitorConfigurer, string, int) {
	count := 0
	var block v1.VisitorConfigurer
	var blockType string

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

func IsVisitorType(typ string) bool {
	switch typ {
	case "stcp", "sudp", "xtcp":
		return true
	default:
		return false
	}
}
