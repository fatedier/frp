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

	"github.com/vaughan0/go-ini"
)

type baseConfig struct {
	// AuthenticationMethod specifies what authentication method to use to
	// authenticate frpc with frps. If "token" is specified - token will be
	// read into login message. If "oidc" is specified - OIDC (Open ID Connect)
	// token will be issued using OIDC settings. By default, this value is "token".
	AuthenticationMethod string `json:"authentication_method"`
	// AuthenticateHeartBeats specifies whether to include authentication token in
	// heartbeats sent to frps. By default, this value is false.
	AuthenticateHeartBeats bool `json:"authenticate_heartbeats"`
	// AuthenticateNewWorkConns specifies whether to include authentication token in
	// new work connections sent to frps. By default, this value is false.
	AuthenticateNewWorkConns bool `json:"authenticate_new_work_conns"`
}

func getDefaultBaseConf() baseConfig {
	return baseConfig{
		AuthenticationMethod:     "token",
		AuthenticateHeartBeats:   false,
		AuthenticateNewWorkConns: false,
	}
}

func unmarshalBaseConfFromIni(conf ini.File) baseConfig {
	var (
		tmpStr string
		ok     bool
	)

	cfg := getDefaultBaseConf()

	if tmpStr, ok = conf.Get("common", "authentication_method"); ok {
		cfg.AuthenticationMethod = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "authenticate_heartbeats"); ok && tmpStr == "true" {
		cfg.AuthenticateHeartBeats = true
	} else {
		cfg.AuthenticateHeartBeats = false
	}

	if tmpStr, ok = conf.Get("common", "authenticate_new_work_conns"); ok && tmpStr == "true" {
		cfg.AuthenticateNewWorkConns = true
	} else {
		cfg.AuthenticateNewWorkConns = false
	}

	return cfg
}

type ClientConfig struct {
	baseConfig
	oidcClientConfig
	tokenConfig
}

func GetDefaultClientConf() ClientConfig {
	return ClientConfig{
		baseConfig:       getDefaultBaseConf(),
		oidcClientConfig: getDefaultOidcClientConf(),
		tokenConfig:      getDefaultTokenConf(),
	}
}

func UnmarshalClientConfFromIni(conf ini.File) (cfg ClientConfig) {
	cfg.baseConfig = unmarshalBaseConfFromIni(conf)
	cfg.oidcClientConfig = unmarshalOidcClientConfFromIni(conf)
	cfg.tokenConfig = unmarshalTokenConfFromIni(conf)
	return cfg
}

type ServerConfig struct {
	baseConfig
	oidcServerConfig
	tokenConfig
}

func GetDefaultServerConf() ServerConfig {
	return ServerConfig{
		baseConfig:       getDefaultBaseConf(),
		oidcServerConfig: getDefaultOidcServerConf(),
		tokenConfig:      getDefaultTokenConf(),
	}
}

func UnmarshalServerConfFromIni(conf ini.File) (cfg ServerConfig) {
	cfg.baseConfig = unmarshalBaseConfFromIni(conf)
	cfg.oidcServerConfig = unmarshalOidcServerConfFromIni(conf)
	cfg.tokenConfig = unmarshalTokenConfFromIni(conf)
	return cfg
}

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

func NewAuthSetter(cfg ClientConfig) (authProvider Setter) {
	switch cfg.AuthenticationMethod {
	case consts.TokenAuthMethod:
		authProvider = NewTokenAuth(cfg.baseConfig, cfg.tokenConfig)
	case consts.OidcAuthMethod:
		authProvider = NewOidcAuthSetter(cfg.baseConfig, cfg.oidcClientConfig)
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
		authVerifier = NewTokenAuth(cfg.baseConfig, cfg.tokenConfig)
	case consts.OidcAuthMethod:
		authVerifier = NewOidcAuthVerifier(cfg.baseConfig, cfg.oidcServerConfig)
	}

	return authVerifier
}
