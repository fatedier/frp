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

	"github.com/fatedier/frp/models/msg"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2/clientcredentials"
)

type OidcAuthProvider struct {
	tokenGenerator         *clientcredentials.Config
	authenticateHeartBeats bool
}

func NewOidcAuthSetter(clientId string, clientSecret string, audience string, tokenEndpointUrl string, authenticateHeartBeats bool) *OidcAuthProvider {
	tokenGenerator := &clientcredentials.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{audience},
		TokenURL:     tokenEndpointUrl,
	}

	return &OidcAuthProvider{
		tokenGenerator:         tokenGenerator,
		authenticateHeartBeats: authenticateHeartBeats,
	}
}

func (auth *OidcAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	tokenObj, err := auth.tokenGenerator.Token(context.Background())
	if tokenObj == nil {
		return fmt.Errorf("couldn't generate OIDC token for login: %s", err)
	}
	loginMsg.PrivilegeKey = tokenObj.AccessToken
	return
}

func (auth *OidcAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !auth.authenticateHeartBeats {
		// if heartbeat authentication is disabled - don't set
		return nil
	}

	tokenObj, err := auth.tokenGenerator.Token(context.Background())
	if tokenObj == nil {
		return fmt.Errorf("couldn't generate OIDC token for ping: %s", err)
	}
	pingMsg.PrivilegeKey = tokenObj.AccessToken
	return
}

type OidcAuthConsumer struct {
	verifier               *oidc.IDTokenVerifier
	authenticateHeartBeats bool
	subjectFromLogin       string
}

func NewOidcAuthVerifier(issuer string, audience string, skipExpiryCheck bool, skipIssuerCheck bool, authenticateHeartBeats bool) *OidcAuthConsumer {
	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		panic(err)
	}
	verifierConf := oidc.Config{
		ClientID:          audience,
		SkipClientIDCheck: audience == "",
		SkipExpiryCheck:   skipExpiryCheck,
		SkipIssuerCheck:   skipIssuerCheck,
	}
	return &OidcAuthConsumer{
		verifier:               provider.Verifier(&verifierConf),
		authenticateHeartBeats: authenticateHeartBeats,
	}
}

func (auth *OidcAuthConsumer) VerifyLogin(loginMsg *msg.Login) (err error) {
	token, err := auth.verifier.Verify(context.Background(), loginMsg.PrivilegeKey)
	if token != nil {
		auth.subjectFromLogin = token.Subject
		return
	}
	return fmt.Errorf("invalid OIDC token in login: %v", err)
}

func (auth *OidcAuthConsumer) VerifyPing(pingMsg *msg.Ping) (err error) {
	if !auth.authenticateHeartBeats {
		// if heartbeat authentication is disabled - don't verify
		return nil
	}

	token, err := auth.verifier.Verify(context.Background(), pingMsg.PrivilegeKey)
	if token == nil {
		return fmt.Errorf("invalid OIDC token in ping: %v", err)
	}
	if token.Subject != auth.subjectFromLogin {
		return fmt.Errorf("received different OIDC subject in login and ping. "+
			"original subject: %s, "+
			"new subject: %s",
			auth.subjectFromLogin, token.Subject)
	}
	return
}
