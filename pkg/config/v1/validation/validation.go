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
	"errors"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	splugin "github.com/fatedier/frp/pkg/plugin/server"
)

var (
	SupportedTransportProtocols = []string{
		"tcp",
		"kcp",
		"quic",
		"websocket",
		"wss",
	}

	SupportedAuthMethods = []v1.AuthMethod{
		"token",
		"oidc",
	}

	SupportedAuthAdditionalScopes = []v1.AuthScope{
		"HeartBeats",
		"NewWorkConns",
	}

	SupportedLogLevels = []string{
		"trace",
		"debug",
		"info",
		"warn",
		"error",
	}

	SupportedHTTPPluginOps = []string{
		splugin.OpLogin,
		splugin.OpNewProxy,
		splugin.OpCloseProxy,
		splugin.OpPing,
		splugin.OpNewWorkConn,
		splugin.OpNewUserConn,
	}
)

type Warning error

func AppendError(err error, errs ...error) error {
	if len(errs) == 0 {
		return err
	}
	return errors.Join(append([]error{err}, errs...)...)
}
