// Copyright 2023 The frp Authors
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

package validation

import (
	"fmt"
	"slices"

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func (v *ConfigValidator) ValidateServerConfig(c *v1.ServerConfig) (Warning, error) {
	var (
		warnings Warning
		errs     error
	)
	if !slices.Contains(SupportedAuthMethods, c.Auth.Method) {
		errs = AppendError(errs, fmt.Errorf("invalid auth method, optional values are %v", SupportedAuthMethods))
	}
	if !lo.Every(SupportedAuthAdditionalScopes, c.Auth.AdditionalScopes) {
		errs = AppendError(errs, fmt.Errorf("invalid auth additional scopes, optional values are %v", SupportedAuthAdditionalScopes))
	}

	errs = AppendError(errs, v.validateAuthTokenSource(c.Auth.Token, c.Auth.TokenSource))

	if err := validateLogConfig(&c.Log); err != nil {
		errs = AppendError(errs, err)
	}

	if err := validateWebServerConfig(&c.WebServer); err != nil {
		errs = AppendError(errs, err)
	}
	if c.Transport.Protocol != "" && !slices.Contains(SupportedTransportProtocols, c.Transport.Protocol) {
		errs = AppendError(errs, fmt.Errorf("invalid transport.protocol, optional values are %v", SupportedTransportProtocols))
	}
	if c.Transport.Protocol == v1.TransportProtocolAuto && !lo.FromPtr(c.Transport.Auto.Enabled) {
		errs = AppendError(errs, fmt.Errorf("transport.auto.enabled must be true when transport.protocol is auto"))
	}
	if c.Transport.Protocol == v1.TransportProtocolAuto && lo.FromPtr(c.Transport.Auto.Enabled) {
		if c.BindPort == 0 {
			errs = AppendError(errs, fmt.Errorf("bindPort must be configured when transport.protocol is auto"))
		}
		if c.KCPBindPort > 0 && c.KCPBindPort != c.BindPort {
			errs = AppendError(errs, fmt.Errorf("kcpBindPort must equal bindPort when transport.protocol is auto"))
		}
		if c.QUICBindPort > 0 && c.QUICBindPort == c.BindPort {
			errs = AppendError(errs, fmt.Errorf("quicBindPort must be different from bindPort when transport.protocol is auto"))
		}
		if c.QUICBindPort > 0 && c.KCPBindPort > 0 && c.QUICBindPort == c.KCPBindPort {
			errs = AppendError(errs, fmt.Errorf("quicBindPort must be different from kcpBindPort when transport.protocol is auto"))
		}
		if err := validateProtocolList("transport.auto.advertiseProtocols", c.Transport.Auto.AdvertiseProtocols); err != nil {
			errs = AppendError(errs, err)
		}
		if slices.Contains(c.Transport.Auto.AdvertiseProtocols, v1.TransportProtocolTCP) && c.BindPort <= 0 {
			errs = AppendError(errs, fmt.Errorf("bindPort must be configured when tcp is in transport.auto.advertiseProtocols"))
		}
		if slices.Contains(c.Transport.Auto.AdvertiseProtocols, v1.TransportProtocolKCP) && c.KCPBindPort <= 0 {
			errs = AppendError(errs, fmt.Errorf("kcpBindPort must be configured when kcp is in transport.auto.advertiseProtocols"))
		}
		if slices.Contains(c.Transport.Auto.AdvertiseProtocols, v1.TransportProtocolQUIC) && c.QUICBindPort <= 0 {
			errs = AppendError(errs, fmt.Errorf("quicBindPort must be configured when quic is in transport.auto.advertiseProtocols"))
		}
		if (slices.Contains(c.Transport.Auto.AdvertiseProtocols, v1.TransportProtocolWebsocket) ||
			slices.Contains(c.Transport.Auto.AdvertiseProtocols, v1.TransportProtocolWSS)) && c.BindPort <= 0 {
			errs = AppendError(errs, fmt.Errorf("bindPort must be configured when websocket/wss is in transport.auto.advertiseProtocols"))
		}
		if c.Transport.Auto.SwitchCooldownSec < 0 {
			errs = AppendError(errs, fmt.Errorf("transport.auto.switchCooldownSec must be non-negative"))
		}
		if err := validateProtocolList("transport.auto.preferOrder", c.Transport.Auto.PreferOrder); err != nil {
			errs = AppendError(errs, err)
		}
	}

	errs = AppendError(errs, ValidatePort(c.BindPort, "bindPort"))
	errs = AppendError(errs, ValidatePort(c.KCPBindPort, "kcpBindPort"))
	errs = AppendError(errs, ValidatePort(c.QUICBindPort, "quicBindPort"))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPPort, "vhostHTTPPort"))
	errs = AppendError(errs, ValidatePort(c.VhostHTTPSPort, "vhostHTTPSPort"))
	errs = AppendError(errs, ValidatePort(c.TCPMuxHTTPConnectPort, "tcpMuxHTTPConnectPort"))
	if c.Transport.MaxPoolCount < 0 {
		errs = AppendError(errs, fmt.Errorf("invalid transport.maxPoolCount, must be non-negative"))
	}

	for _, p := range c.HTTPPlugins {
		if !lo.Every(SupportedHTTPPluginOps, p.Ops) {
			errs = AppendError(errs, fmt.Errorf("invalid http plugin ops, optional values are %v", SupportedHTTPPluginOps))
		}
	}
	return warnings, errs
}
