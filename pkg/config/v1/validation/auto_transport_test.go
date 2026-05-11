// Copyright 2026 The frp Authors
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
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
)

func TestValidateServerAutoTransportPorts(t *testing.T) {
	validator := NewConfigValidator(security.NewUnsafeFeatures(nil))

	t.Run("fixed model is valid", func(t *testing.T) {
		cfg := &v1.ServerConfig{
			BindPort:     7000,
			KCPBindPort:  7000,
			QUICBindPort: 7002,
		}
		cfg.Transport.Protocol = v1.TransportProtocolAuto
		require.NoError(t, cfg.Complete())

		_, err := validator.ValidateServerConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("kcp must equal bind port", func(t *testing.T) {
		cfg := &v1.ServerConfig{
			BindPort:     7000,
			KCPBindPort:  7001,
			QUICBindPort: 7002,
		}
		cfg.Transport.Protocol = v1.TransportProtocolAuto
		require.NoError(t, cfg.Complete())

		_, err := validator.ValidateServerConfig(cfg)
		require.ErrorContains(t, err, "kcpBindPort must equal bindPort")
	})

	t.Run("quic must be independent", func(t *testing.T) {
		cfg := &v1.ServerConfig{
			BindPort:     7000,
			KCPBindPort:  7000,
			QUICBindPort: 7000,
		}
		cfg.Transport.Protocol = v1.TransportProtocolAuto
		require.NoError(t, cfg.Complete())

		_, err := validator.ValidateServerConfig(cfg)
		require.ErrorContains(t, err, "quicBindPort must be different")
	})
}

func TestValidateClientAutoTransportProxyURLAllowed(t *testing.T) {
	validator := NewConfigValidator(security.NewUnsafeFeatures(nil))
	cfg := &v1.ClientCommonConfig{}
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	cfg.Transport.ProxyURL = "http://127.0.0.1:8080"
	require.NoError(t, cfg.Complete())

	_, err := validator.ValidateClientCommonConfig(cfg)
	require.NoError(t, err)
}

func TestValidateClientAutoTransportStrategy(t *testing.T) {
	validator := NewConfigValidator(security.NewUnsafeFeatures(nil))

	cfg := &v1.ClientCommonConfig{}
	cfg.Transport.Protocol = v1.TransportProtocolAuto
	cfg.Transport.Auto.Strategy = v1.AutoTransportStrategyLatency
	require.NoError(t, cfg.Complete())

	_, err := validator.ValidateClientCommonConfig(cfg)
	require.NoError(t, err)

	cfg.Transport.Auto.Strategy = "fastest"
	_, err = validator.ValidateClientCommonConfig(cfg)
	require.ErrorContains(t, err, "invalid transport.auto.strategy")
}
