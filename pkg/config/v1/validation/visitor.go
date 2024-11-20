package validation

import (
	"errors"
	"fmt"
	"slices"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func ValidateVisitorConfigurer(c v1.VisitorConfigurer) error {
	base := c.GetBaseConfig()
	if err := validateVisitorBaseConfig(base); err != nil {
		return err
	}

	switch v := c.(type) {
	case *v1.STCPVisitorConfig:
	case *v1.SUDPVisitorConfig:
	case *v1.XTCPVisitorConfig:
		return validateXTCPVisitorConfig(v)
	default:
		return errors.New("unknown visitor config type")
	}
	return nil
}

func validateVisitorBaseConfig(c *v1.VisitorBaseConfig) error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.ServerName == "" {
		return errors.New("server name is required")
	}

	if c.BindPort == 0 {
		return errors.New("bind port is required")
	}
	return nil
}

func validateXTCPVisitorConfig(c *v1.XTCPVisitorConfig) error {
	if !slices.Contains([]string{"kcp", "quic"}, c.Protocol) {
		return fmt.Errorf("protocol should be kcp or quic")
	}
	return nil
}
