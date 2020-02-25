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

	"github.com/fatedier/frp/models/msg"
	"github.com/fatedier/frp/utils/util"
)

type tokenConfig struct {
	// Token specifies the authorization token used to create keys to be sent
	// to the server. The server must have a matching token for authorization
	// to succeed.  By default, this value is "".
	Token string `json:"token"`
}

func getDefaultTokenConf() tokenConfig {
	return tokenConfig{
		Token: "",
	}
}

type TokenAuthSetterVerifier struct {
	baseConfig

	token string
}

func NewTokenAuth(baseCfg baseConfig, cfg tokenConfig) *TokenAuthSetterVerifier {
	return &TokenAuthSetterVerifier{
		baseConfig: baseCfg,
		token:      cfg.Token,
	}
}

func (auth *TokenAuthSetterVerifier) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey = util.GetAuthKey(auth.token, loginMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) SetPing(*msg.Ping) error {
	// Ping doesn't include authentication in token method
	return nil
}

func (auth *TokenAuthSetterVerifier) SetNewWorkConn(*msg.NewWorkConn) error {
	// NewWorkConn doesn't include authentication in token method
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyLogin(loginMsg *msg.Login) error {
	if util.GetAuthKey(auth.token, loginMsg.Timestamp) != loginMsg.PrivilegeKey {
		return fmt.Errorf("token in login doesn't match token from configuration")
	}
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyPing(*msg.Ping) error {
	// Ping doesn't include authentication in token method
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyNewWorkConn(*msg.NewWorkConn) error {
	// NewWorkConn doesn't include authentication in token method
	return nil
}
