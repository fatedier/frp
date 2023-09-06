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

package legacy

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
	// if AuthenticationMethod == "oidc". By default, this value is "".
	OidcAudience string `ini:"oidc_audience" json:"oidc_audience"`
	// OidcScope specifies the scope of the token in OIDC authentication
	// if AuthenticationMethod == "oidc". By default, this value is "".
	OidcScope string `ini:"oidc_scope" json:"oidc_scope"`
	// OidcTokenEndpointURL specifies the URL which implements OIDC Token Endpoint.
	// It will be used to get an OIDC token if AuthenticationMethod == "oidc".
	// By default, this value is "".
	OidcTokenEndpointURL string `ini:"oidc_token_endpoint_url" json:"oidc_token_endpoint_url"`

	// OidcAdditionalEndpointParams specifies additional parameters to be sent
	// this field will be transfer to map[string][]string in OIDC token generator
	// The field will be set by prefix "oidc_additional_"
	OidcAdditionalEndpointParams map[string]string `ini:"-" json:"oidc_additional_endpoint_params"`
}

func getDefaultOidcClientConf() OidcClientConfig {
	return OidcClientConfig{
		OidcClientID:                 "",
		OidcClientSecret:             "",
		OidcAudience:                 "",
		OidcScope:                    "",
		OidcTokenEndpointURL:         "",
		OidcAdditionalEndpointParams: make(map[string]string),
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

type TokenConfig struct {
	// Token specifies the authorization token used to create keys to be sent
	// to the server. The server must have a matching token for authorization
	// to succeed.  By default, this value is "".
	Token string `ini:"token" json:"token"`
}

func getDefaultTokenConf() TokenConfig {
	return TokenConfig{
		Token: "",
	}
}
