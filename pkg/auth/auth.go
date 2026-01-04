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
	"context"
	"fmt"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type Setter interface {
	SetLogin(*msg.Login) error
	SetPing(*msg.Ping) error
	SetNewWorkConn(*msg.NewWorkConn) error
}

type ClientAuth struct {
	Setter Setter
	key    []byte
}

func (a *ClientAuth) EncryptionKey() []byte {
	return a.key
}

// BuildClientAuth resolves any dynamic auth values and returns a prepared auth runtime.
// Caller must run validation before calling this function.
func BuildClientAuth(cfg *v1.AuthClientConfig) (*ClientAuth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("auth config is nil")
	}
	resolved := *cfg
	if resolved.Method == v1.AuthMethodToken && resolved.TokenSource != nil {
		token, err := resolved.TokenSource.Resolve(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to resolve auth.tokenSource: %w", err)
		}
		resolved.Token = token
	}
	setter, err := NewAuthSetter(resolved)
	if err != nil {
		return nil, err
	}
	return &ClientAuth{
		Setter: setter,
		key:    []byte(resolved.Token),
	}, nil
}

func NewAuthSetter(cfg v1.AuthClientConfig) (authProvider Setter, err error) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authProvider = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodOIDC:
		if cfg.OIDC.TokenSource != nil {
			authProvider = NewOidcTokenSourceAuthSetter(cfg.AdditionalScopes, cfg.OIDC.TokenSource)
		} else {
			authProvider, err = NewOidcAuthSetter(cfg.AdditionalScopes, cfg.OIDC)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", cfg.Method)
	}
	return authProvider, nil
}

type Verifier interface {
	VerifyLogin(*msg.Login) error
	VerifyPing(*msg.Ping) error
	VerifyNewWorkConn(*msg.NewWorkConn) error
}

type ServerAuth struct {
	Verifier Verifier
	key      []byte
}

func (a *ServerAuth) EncryptionKey() []byte {
	return a.key
}

// BuildServerAuth resolves any dynamic auth values and returns a prepared auth runtime.
// Caller must run validation before calling this function.
func BuildServerAuth(cfg *v1.AuthServerConfig) (*ServerAuth, error) {
	if cfg == nil {
		return nil, fmt.Errorf("auth config is nil")
	}
	resolved := *cfg
	if resolved.Method == v1.AuthMethodToken && resolved.TokenSource != nil {
		token, err := resolved.TokenSource.Resolve(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to resolve auth.tokenSource: %w", err)
		}
		resolved.Token = token
	}
	return &ServerAuth{
		Verifier: NewAuthVerifier(resolved),
		key:      []byte(resolved.Token),
	}, nil
}

func NewAuthVerifier(cfg v1.AuthServerConfig) (authVerifier Verifier) {
	switch cfg.Method {
	case v1.AuthMethodToken:
		authVerifier = NewTokenAuth(cfg.AdditionalScopes, cfg.Token)
	case v1.AuthMethodOIDC:
		tokenVerifier := NewTokenVerifier(cfg.OIDC)
		authVerifier = NewOidcAuthVerifier(cfg.AdditionalScopes, tokenVerifier)
	}
	return authVerifier
}
