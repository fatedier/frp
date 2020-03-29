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
	"github.com/vaughan0/go-ini"
	"golang.org/x/oauth2/clientcredentials"
)

type oidcClientConfig struct {
	// OidcClientId specifies the client ID to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientId string `json:"oidc_client_id"`
	// OidcClientSecret specifies the client secret to use to get a token in OIDC
	// authentication if AuthenticationMethod == "oidc". By default, this value
	// is "".
	OidcClientSecret string `json:"oidc_client_secret"`
	// OidcAudience specifies the audience of the token in OIDC authentication
	//if AuthenticationMethod == "oidc". By default, this value is "".
	OidcAudience string `json:"oidc_audience"`
	// OidcTokenEndpointUrl specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token if AuthenticationMethod == "oidc".
	// By default, this value is "".
	OidcTokenEndpointUrl string `json:"oidc_token_endpoint_url"`
}

func getDefaultOidcClientConf() oidcClientConfig {
	return oidcClientConfig{
		OidcClientId:         "",
		OidcClientSecret:     "",
		OidcAudience:         "",
		OidcTokenEndpointUrl: "",
	}
}

func unmarshalOidcClientConfFromIni(conf ini.File) oidcClientConfig {
	var (
		tmpStr string
		ok     bool
	)

	cfg := getDefaultOidcClientConf()

	if tmpStr, ok = conf.Get("common", "oidc_client_id"); ok {
		cfg.OidcClientId = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "oidc_client_secret"); ok {
		cfg.OidcClientSecret = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "oidc_audience"); ok {
		cfg.OidcAudience = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "oidc_token_endpoint_url"); ok {
		cfg.OidcTokenEndpointUrl = tmpStr
	}

	return cfg
}

type oidcServerConfig struct {
	// OidcIssuer specifies the issuer to verify OIDC tokens with. This issuer
	// will be used to load public keys to verify signature and will be compared
	// with the issuer claim in the OIDC token. It will be used if
	// AuthenticationMethod == "oidc". By default, this value is "".
	OidcIssuer string `json:"oidc_issuer"`
	// OidcAudience specifies the audience OIDC tokens should contain when validated.
	// If this value is empty, audience ("client ID") verification will be skipped.
	// It will be used when AuthenticationMethod == "oidc". By default, this
	// value is "".
	OidcAudience string `json:"oidc_audience"`
	// OidcSkipExpiryCheck specifies whether to skip checking if the OIDC token is
	// expired. It will be used when AuthenticationMethod == "oidc". By default, this
	// value is false.
	OidcSkipExpiryCheck bool `json:"oidc_skip_expiry_check"`
	// OidcSkipIssuerCheck specifies whether to skip checking if the OIDC token's
	// issuer claim matches the issuer specified in OidcIssuer. It will be used when
	// AuthenticationMethod == "oidc". By default, this value is false.
	OidcSkipIssuerCheck bool `json:"oidc_skip_issuer_check"`
}

func getDefaultOidcServerConf() oidcServerConfig {
	return oidcServerConfig{
		OidcIssuer:          "",
		OidcAudience:        "",
		OidcSkipExpiryCheck: false,
		OidcSkipIssuerCheck: false,
	}
}

func unmarshalOidcServerConfFromIni(conf ini.File) oidcServerConfig {
	var (
		tmpStr string
		ok     bool
	)

	cfg := getDefaultOidcServerConf()

	if tmpStr, ok = conf.Get("common", "oidc_issuer"); ok {
		cfg.OidcIssuer = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "oidc_audience"); ok {
		cfg.OidcAudience = tmpStr
	}

	if tmpStr, ok = conf.Get("common", "oidc_skip_expiry_check"); ok && tmpStr == "true" {
		cfg.OidcSkipExpiryCheck = true
	} else {
		cfg.OidcSkipExpiryCheck = false
	}

	if tmpStr, ok = conf.Get("common", "oidc_skip_issuer_check"); ok && tmpStr == "true" {
		cfg.OidcSkipIssuerCheck = true
	} else {
		cfg.OidcSkipIssuerCheck = false
	}

	return cfg
}

type OidcAuthProvider struct {
	baseConfig

	tokenGenerator *clientcredentials.Config
}

func NewOidcAuthSetter(baseCfg baseConfig, cfg oidcClientConfig) *OidcAuthProvider {
	tokenGenerator := &clientcredentials.Config{
		ClientID:     cfg.OidcClientId,
		ClientSecret: cfg.OidcClientSecret,
		Scopes:       []string{cfg.OidcAudience},
		TokenURL:     cfg.OidcTokenEndpointUrl,
	}

	return &OidcAuthProvider{
		baseConfig:     baseCfg,
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
	baseConfig

	verifier         *oidc.IDTokenVerifier
	subjectFromLogin string
}

func NewOidcAuthVerifier(baseCfg baseConfig, cfg oidcServerConfig) *OidcAuthConsumer {
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
		baseConfig: baseCfg,
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
