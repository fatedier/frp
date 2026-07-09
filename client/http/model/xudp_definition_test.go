package model

import (
	"testing"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestProxyDefinitionNewTypesRoundTrip(t *testing.T) {
	xudp := &v1.XUDPProxyConfig{}
	xudp.Name, xudp.Type, xudp.Secretkey, xudp.LocalPort = "u", "xudp", "k", 3389

	combo := &v1.XTCPXUDPProxyConfig{}
	combo.Name, combo.Type, combo.Secretkey, combo.LocalPort = "c", "xtcp+xudp", "k", 3389

	cases := []struct {
		cfg     v1.ProxyConfigurer
		typ     string
		isXUDP  bool
		isCombo bool
	}{
		{xudp, "xudp", true, false},
		{combo, "xtcp+xudp", false, true},
	}
	for _, tc := range cases {
		def, err := ProxyDefinitionFromConfigurer(tc.cfg)
		if err != nil {
			t.Fatalf("%s FromConfigurer: %v", tc.typ, err)
		}
		if def.Type != tc.typ {
			t.Fatalf("%s: type mismatch got %q", tc.typ, def.Type)
		}
		if tc.isXUDP && def.XUDP == nil {
			t.Fatalf("%s: XUDP block missing", tc.typ)
		}
		if tc.isCombo && def.XTCPXUDP == nil {
			t.Fatalf("%s: XTCPXUDP block missing", tc.typ)
		}
		if !IsProxyType(tc.typ) {
			t.Fatalf("%s: IsProxyType false", tc.typ)
		}
		if err := def.Validate("", false); err != nil {
			t.Fatalf("%s Validate: %v", tc.typ, err)
		}
		back, err := def.ToConfigurer()
		if err != nil {
			t.Fatalf("%s ToConfigurer: %v", tc.typ, err)
		}
		if back.GetBaseConfig().Type != tc.typ {
			t.Fatalf("%s: round-trip type mismatch %q", tc.typ, back.GetBaseConfig().Type)
		}
	}
}

func TestVisitorDefinitionNewTypesRoundTrip(t *testing.T) {
	xudp := &v1.XUDPVisitorConfig{}
	combo := &v1.XTCPXUDPVisitorConfig{}

	cases := []struct {
		cfg v1.VisitorConfigurer
		typ string
	}{
		{xudp, "xudp"},
		{combo, "xtcp+xudp"},
	}
	for _, tc := range cases {
		base := tc.cfg.GetBaseConfig()
		base.Name, base.Type, base.ServerName, base.BindPort = "v", tc.typ, "srv", 13389

		def, err := VisitorDefinitionFromConfigurer(tc.cfg)
		if err != nil {
			t.Fatalf("%s FromConfigurer: %v", tc.typ, err)
		}
		if def.Type != tc.typ {
			t.Fatalf("%s: type mismatch got %q", tc.typ, def.Type)
		}
		if !IsVisitorType(tc.typ) {
			t.Fatalf("%s: IsVisitorType false", tc.typ)
		}
		if err := def.Validate("", false); err != nil {
			t.Fatalf("%s Validate: %v", tc.typ, err)
		}
		if _, err := def.ToConfigurer(); err != nil {
			t.Fatalf("%s ToConfigurer: %v", tc.typ, err)
		}
	}
}
