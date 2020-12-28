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

	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/msg"
)

type BaseConfig struct {
	// AuthenticationMethod specifies what authentication method to use to
	// authenticate frpc with frps. If "token" is specified - token will be
	// read into login message. If "oidc" is specified - OIDC (Open ID Connect)
	// token will be issued using OIDC settings. By default, this value is "token".
	AuthenticationMethod string `ini:"authentication_method" json:"authentication_method"`
	// AuthenticateHeartBeats specifies whether to include authentication token in
	// heartbeats sent to frps. By default, this value is false.
	AuthenticateHeartBeats bool `ini:"authenticate_heartbeats" json:"authenticate_heartbeats"`
	// AuthenticateNewWorkConns specifies whether to include authentication token in
	// new work connections sent to frps. By default, this value is false.
	AuthenticateNewWorkConns bool `ini:"authenticate_new_work_conns" json:"authenticate_new_work_conns"`
}

func getDefaultBaseConf() BaseConfig {
	return BaseConfig{
		AuthenticationMethod:     "token",
		AuthenticateHeartBeats:   false,
		AuthenticateNewWorkConns: false,
	}
}

type ClientConfig struct {
	BaseConfig       `ini:",extends"`
	OidcClientConfig `ini:",extends"`
	TokenConfig      `ini:",extends"`
}

func GetDefaultClientConf() ClientConfig {
	return ClientConfig{
		BaseConfig:       getDefaultBaseConf(),
		OidcClientConfig: getDefaultOidcClientConf(),
		TokenConfig:      getDefaultTokenConf(),
	}
}

type ServerConfig struct {
	BaseConfig       `ini:",extends"`
	OidcServerConfig `ini:",extends"`
	TokenConfig      `ini:",extends"`
}

func GetDefaultServerConf() ServerConfig {
	return ServerConfig{
		BaseConfig:       getDefaultBaseConf(),
		OidcServerConfig: getDefaultOidcServerConf(),
		TokenConfig:      getDefaultTokenConf(),
	}
}

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthSetter(cfg ClientConfig) (authProvider Setter) {
	switch cfg.AuthenticationMethod {
	case consts.TokenAuthMethod:
		authProvider = NewTokenAuth(cfg.BaseConfig, cfg.TokenConfig)
	case consts.OidcAuthMethod:
		authProvider = NewOidcAuthSetter(cfg.BaseConfig, cfg.OidcClientConfig)
	default:
		panic(fmt.Sprintf("wrong authentication method: '%s'", cfg.AuthenticationMethod))
	}

	return authProvider
}

type Verifier interface {
	VerifyLogin(*msg.Login) error
	VerifyPing(*msg.Ping) error
	VerifyNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthVerifier(cfg ServerConfig) (authVerifier Verifier) {
	switch cfg.AuthenticationMethod {
	case consts.TokenAuthMethod:
		authVerifier = NewTokenAuth(cfg.BaseConfig, cfg.TokenConfig)
	case consts.OidcAuthMethod:
		authVerifier = NewOidcAuthVerifier(cfg.BaseConfig, cfg.OidcServerConfig)
	}

	return authVerifier
}
