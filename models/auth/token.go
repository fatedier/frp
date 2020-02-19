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

type TokenAuthProviderConsumer struct {
	Token string
}

func NewTokenAuth(token string) *TokenAuthProviderConsumer {
	return &TokenAuthProviderConsumer{
		Token: token,
	}
}

func (auth *TokenAuthProviderConsumer) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey = util.GetAuthKey(auth.Token, loginMsg.Timestamp)
	return nil
}

func (auth *TokenAuthProviderConsumer) SetPing(*msg.Ping) error {
	// ping doesn't include authentication in token method
	return nil
}

type TokenAuthConsumer struct {
	Token string
}

func (auth *TokenAuthProviderConsumer) VerifyLogin(loginMsg *msg.Login) error {
	if util.GetAuthKey(auth.Token, loginMsg.Timestamp) != loginMsg.PrivilegeKey {
		return fmt.Errorf("token in login doesn't match token from configuration")
	}
	return nil
}

func (auth *TokenAuthProviderConsumer) VerifyPing(*msg.Ping) error {
	// ping doesn't include authentication in token method
	return nil
}
