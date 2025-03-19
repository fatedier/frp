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
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthSetter(cfg v1.AuthClientConfig) (authProvider Setter) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authProvider = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodJWT:
		authProvider = NewJWTAuth(cfg.AdditionalScopes, cfg.Token, cfg.Secret)
	case v1.AuthMethodOIDC:
		authProvider = NewOidcAuthSetter(cfg.AdditionalScopes, cfg.OIDC)
	default:
		panic(fmt.Sprintf("wrong method: '%s'", cfg.Method))
	}
	return authProvider
}

type Verifier interface {
	VerifyLogin(*msg.Login) error
	VerifyPing(*msg.Ping) error
	VerifyNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthVerifier(cfg v1.AuthServerConfig) (authVerifier Verifier) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authVerifier = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodJWT:
		authVerifier = NewJWTAuth(cfg.AdditionalScopes, cfg.Token, cfg.Secret)
	case v1.AuthMethodOIDC:
		tokenVerifier := NewTokenVerifier(cfg.OIDC)
		authVerifier = NewOidcAuthVerifier(cfg.AdditionalScopes, tokenVerifier)
	}
	return authVerifier
}
