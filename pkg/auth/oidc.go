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

	"github.com/fatedier/frp/pkg/msg"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2/clientcredentials"
)

type OidcClientConfig struct {
	// OidcClientID specifies the client ID to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientID string `ini:"oidc_client_id" json:"oidc_client_id"`
	// OidcClientSecret specifies the client secret to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientSecret string `ini:"oidc_client_secret" json:"oidc_client_secret"`
	// OidcAudience specifies the audience of the token in OIDC authentication
	//if AuthenticationMethod == "oidc". By default, this value is "".
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcTokenEndpointURL specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token if AuthenticationMethod == "oidc".
	// By default, this value is "".
	OidcTokenEndpointURL string `ini:"oidc_token_endpoint_url" json:"oidc_token_endpoint_url"`
}

func getDefaultOidcClientConf() OidcClientConfig {
	return OidcClientConfig{
		OidcClientID:         "",
		OidcClientSecret:     "",
		OidcAudience:         "",
		OidcTokenEndpointURL: "",
	}
}

type OidcServerConfig struct {
	// OidcIssuer specifies the issuer to verify OIDC tokens with. This issuer
	// will be used to load public keys to verify signature and will be compared
	// with the issuer claim in the OIDC token. It will be used if
	// AuthenticationMethod == "oidc". By default, this value is "".
	OidcIssuer string `ini:"oidc_issuer" json:"oidc_issuer"`
	// OidcAudience specifies the audience OIDC tokens should contain when validated.
	// If this value is empty, audience ("client ID") verification will be skipped.
	// It will be used when AuthenticationMethod == "oidc". By default, this
	// value is "".
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcSkipExpiryCheck specifies whether to skip checking if the OIDC token is
	// expired. It will be used when AuthenticationMethod == "oidc". By default, this
	// value is false.
	OidcSkipExpiryCheck bool `ini:"oidc_skip_expiry_check" json:"oidc_skip_expiry_check"`
	// OidcSkipIssuerCheck specifies whether to skip checking if the OIDC token's
	// issuer claim matches the issuer specified in OidcIssuer. It will be used when
	// AuthenticationMethod == "oidc". By default, this value is false.
	OidcSkipIssuerCheck bool `ini:"oidc_skip_issuer_check" json:"oidc_skip_issuer_check"`
}

func getDefaultOidcServerConf() OidcServerConfig {
	return OidcServerConfig{
		OidcIssuer:          "",
		OidcAudience:        "",
		OidcSkipExpiryCheck: false,
		OidcSkipIssuerCheck: false,
	}
}

type OidcAuthProvider struct {
	BaseConfig

	tokenGenerator *clientcredentials.Config
}

func NewOidcAuthSetter(baseCfg BaseConfig, cfg OidcClientConfig) *OidcAuthProvider {
	tokenGenerator := &clientcredentials.Config{
		ClientID:     cfg.OidcClientID,
		ClientSecret: cfg.OidcClientSecret,
		Scopes:       []string{cfg.OidcAudience},
		TokenURL:     cfg.OidcTokenEndpointURL,
	}

	return &OidcAuthProvider{
		BaseConfig:     baseCfg,
		tokenGenerator: tokenGenerator,
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
	if !auth.AuthenticateHeartBeats {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !auth.AuthenticateNewWorkConns {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

type OidcAuthConsumer struct {
	BaseConfig

	verifier         *oidc.IDTokenVerifier
	subjectFromLogin string
}

func NewOidcAuthVerifier(baseCfg BaseConfig, cfg OidcServerConfig) *OidcAuthConsumer {
	provider, err := oidc.NewProvider(context.Background(), cfg.OidcIssuer)
	if err != nil {
		panic(err)
	}
	verifierConf := oidc.Config{
		ClientID:          cfg.OidcAudience,
		SkipClientIDCheck: cfg.OidcAudience == "",
		SkipExpiryCheck:   cfg.OidcSkipExpiryCheck,
		SkipIssuerCheck:   cfg.OidcSkipIssuerCheck,
	}
	return &OidcAuthConsumer{
		BaseConfig: baseCfg,
		verifier:   provider.Verifier(&verifierConf),
	}
}

func (auth *OidcAuthConsumer) VerifyLogin(loginMsg *msg.Login) (err error) {
	token, err := auth.verifier.Verify(context.Background(), loginMsg.PrivilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in login: %v", err)
	}
	auth.subjectFromLogin = token.Subject
	return nil
}

func (auth *OidcAuthConsumer) verifyPostLoginToken(privilegeKey string) (err error) {
	token, err := auth.verifier.Verify(context.Background(), privilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in ping: %v", err)
	}
	if token.Subject != auth.subjectFromLogin {
		return fmt.Errorf("received different OIDC subject in login and ping. "+
			"original subject: %s, "+
			"new subject: %s",
			auth.subjectFromLogin, token.Subject)
	}
	return nil
}

func (auth *OidcAuthConsumer) VerifyPing(pingMsg *msg.Ping) (err error) {
	if !auth.AuthenticateHeartBeats {
		return nil
	}

	return auth.verifyPostLoginToken(pingMsg.PrivilegeKey)
}

func (auth *OidcAuthConsumer) VerifyNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !auth.AuthenticateNewWorkConns {
		return nil
	}

	return auth.verifyPostLoginToken(newWorkConnMsg.PrivilegeKey)
}
