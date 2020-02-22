// Copyright 2020 guylewin, guy@lewin.co.il
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

package auth

import (
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/models/msg"
)

type BaseAuth struct {
	authenticateHeartBeats   bool
	authenticateNewWorkConns bool
}

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthSetter(cfg config.ClientCommonConf) (authProvider Setter) {
	baseAuth := BaseAuth{
		authenticateHeartBeats:   cfg.AuthenticateHeartBeats,
		authenticateNewWorkConns: cfg.AuthenticateNewWorkConns,
	}

	switch cfg.AuthenticationMethod {
	case consts.TokenAuthMethod:
		authProvider = NewTokenAuth(baseAuth, cfg.Token)
	case consts.OidcAuthMethod:
		authProvider = NewOidcAuthSetter(
			baseAuth,
			cfg.OidcClientId,
			cfg.OidcClientSecret,
			cfg.OidcAudience,
			cfg.OidcTokenEndpointUrl,
		)
	}

	return authProvider
}

type Verifier interface {
	VerifyLogin(*msg.Login) error
	VerifyPing(*msg.Ping) error
	VerifyNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthVerifier(cfg config.ServerCommonConf) (authVerifier Verifier) {
	baseAuth := BaseAuth{
		authenticateHeartBeats:   cfg.AuthenticateHeartBeats,
		authenticateNewWorkConns: cfg.AuthenticateNewWorkConns,
	}

	switch cfg.AuthenticationMethod {
	case consts.TokenAuthMethod:
		authVerifier = NewTokenAuth(baseAuth, cfg.Token)
	case consts.OidcAuthMethod:
		authVerifier = NewOidcAuthVerifier(
			baseAuth,
			cfg.OidcIssuer,
			cfg.OidcAudience,
			cfg.OidcSkipExpiryCheck,
			cfg.OidcSkipIssuerCheck,
		)
	}

	return authVerifier
}
