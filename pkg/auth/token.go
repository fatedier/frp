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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

type TokenAuthSetterVerifier struct {
	additionalAuthScopes []v1.AuthScope
	token                string
}

func NewTokenAuth(additionalAuthScopes []v1.AuthScope, token string) *TokenAuthSetterVerifier {
	return &TokenAuthSetterVerifier{
		additionalAuthScopes: additionalAuthScopes,
		token:                token,
	}
}

func (auth *TokenAuthSetterVerifier) SetLogin(loginMsg *msg.Login) error {
	loginMsg.PrivilegeKey = util.GetAuthKey(auth.token, loginMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) SetPing(pingMsg *msg.Ping) error {
	if !lo.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.Timestamp = time.Now().Unix()
	pingMsg.PrivilegeKey = util.GetAuthKey(auth.token, pingMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) error {
	if !lo.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.Timestamp = time.Now().Unix()
	newWorkConnMsg.PrivilegeKey = util.GetAuthKey(auth.token, newWorkConnMsg.Timestamp)
	return nil
}

func extractBetweenDots(input string) string {
	firstDotIndex := strings.Index(input, ".")
	if firstDotIndex == -1 {
		return ""
	}

	secondDotIndex := strings.Index(input[firstDotIndex+1:], ".")
	if secondDotIndex == -1 {
		return ""
	}

	extracted := input[firstDotIndex+1 : firstDotIndex+1+secondDotIndex]

	return extracted
}
func validateApiKey(apiKey string) bool {
	accoundId := extractBetweenDots(apiKey)

	if accoundId == "" {
		fmt.Errorf("token in login doesn't match format. Can't extract account ID")
		return false
	}

	reqUrl := "https://app.harness.io/authz/api/acl"

	var formated = fmt.Sprintf(`{
    "permissions": [
      {
        "resourceScope": {
          "accountIdentifier": "%s",
          "orgIdentifier": "",
          "projectIdentifier": ""
        },
        "resourceType": "PIPLINE",
        "permission": "core_pipeline_view"
      }
    ]
  	}`, accoundId)

	req, err := http.NewRequest("POST", reqUrl, bytes.NewReader([]byte(formated)))
	if err != nil {
		fmt.Errorf(err.Error())
		return false
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-api-key", apiKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Errorf("Error calling api call for api key validation")
		return false
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Errorf("Api key validation response error")
		return false
	}

	searchSubstring := `"permitted":true`
	bodyStr := string(body)

	if !strings.Contains(bodyStr, searchSubstring) {
		fmt.Errorf("The API key is not valid")
		return false
	}

	return true
}

func (auth *TokenAuthSetterVerifier) VerifyLogin(m *msg.Login) error {
	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in login doesn't match token from configuration")
	}

	if !validateApiKey(m.ApiKey) {
		return fmt.Errorf("Harness Api key isn't valid")
	}

	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyPing(m *msg.Ping) error {
	if !lo.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in heartbeat doesn't match token from configuration")
	}
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyNewWorkConn(m *msg.NewWorkConn) error {
	if !lo.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in NewWorkConn doesn't match token from configuration")
	}
	return nil
}
