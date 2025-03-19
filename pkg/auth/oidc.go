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
	"slices"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2/clientcredentials"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type OidcAuthProvider struct {
	additionalAuthScopes []v1.AuthScope

	tokenGenerator *clientcredentials.Config
}

func NewOidcAuthSetter(additionalAuthScopes []v1.AuthScope, cfg v1.AuthOIDCClientConfig) *OidcAuthProvider {
	eps := make(map[string][]string)
	for k, v := range cfg.AdditionalEndpointParams {
		eps[k] = []string{v}
	}

	if cfg.Audience != "" {
		eps["audience"] = []string{cfg.Audience}
	}

	tokenGenerator := &clientcredentials.Config{
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		Scopes:         []string{cfg.Scope},
		TokenURL:       cfg.TokenEndpointURL,
		EndpointParams: eps,
	}

	return &OidcAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		tokenGenerator:       tokenGenerator,
	}
}

func (auth *OidcAuthProvider) generateAccessToken() (accessToken string, err error) {
	tokenObj, err := auth.tokenGenerator.Token(context.Background())
	if err != nil {
		return "", fmt.Errorf("couldn't generate OIDC token for login: %v", err)
	}
	return tokenObj.AccessToken, nil
}

func (auth *OidcAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

type TokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type OidcAuthConsumer struct {
	additionalAuthScopes []v1.AuthScope

	verifier          TokenVerifier
	subjectsFromLogin []string
}

func NewTokenVerifier(cfg v1.AuthOIDCServerConfig) TokenVerifier {
	provider, err := oidc.NewProvider(context.Background(), cfg.Issuer)
	if err != nil {
		panic(err)
	}
	verifierConf := oidc.Config{
		ClientID:          cfg.Audience,
		SkipClientIDCheck: cfg.Audience == "",
		SkipExpiryCheck:   cfg.SkipExpiryCheck,
		SkipIssuerCheck:   cfg.SkipIssuerCheck,
	}
	return provider.Verifier(&verifierConf)
}

func NewOidcAuthVerifier(additionalAuthScopes []v1.AuthScope, verifier TokenVerifier) *OidcAuthConsumer {
	return &OidcAuthConsumer{
		additionalAuthScopes: additionalAuthScopes,
		verifier:             verifier,
		subjectsFromLogin:    []string{},
	}
}

func (auth *OidcAuthConsumer) VerifyLogin(loginMsg *msg.Login) (err error) {
	token, err := auth.verifier.Verify(context.Background(), loginMsg.PrivilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in login: %v", err)
	}
	if !slices.Contains(auth.subjectsFromLogin, token.Subject) {
		auth.subjectsFromLogin = append(auth.subjectsFromLogin, token.Subject)
	}
	return nil
}

func (auth *OidcAuthConsumer) verifyPostLoginToken(privilegeKey string) (err error) {
	token, err := auth.verifier.Verify(context.Background(), privilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in ping: %v", err)
	}
	if !slices.Contains(auth.subjectsFromLogin, token.Subject) {
		return fmt.Errorf("received different OIDC subject in login and ping. "+
			"original subjects: %s, "+
			"new subject: %s",
			auth.subjectsFromLogin, token.Subject)
	}
	return nil
}

func (auth *OidcAuthConsumer) VerifyPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	return auth.verifyPostLoginToken(pingMsg.PrivilegeKey)
}

func (auth *OidcAuthConsumer) VerifyNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	return auth.verifyPostLoginToken(newWorkConnMsg.PrivilegeKey)
}
