// Copyright 2025 LeanCode Sp. z o.o.
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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type EntraIDAuthProvider struct {
	additionalAuthScopes []v1.AuthScope
	cfg                  v1.AuthEntraIDClientConfig
}

func NewEntraIDAuthSetter(additionalAuthScopes []v1.AuthScope, cfg v1.AuthEntraIDClientConfig) *EntraIDAuthProvider {
	return &EntraIDAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		cfg:                  cfg,
	}
}

func (auth *EntraIDAuthProvider) generateAccessToken() (accessToken string, err error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("failed to initialize Azure credential: %w", err)
	}

	scope := fmt.Sprintf("%s/.default", auth.cfg.Audience)
	opts := policy.TokenRequestOptions{
		Scopes:   []string{scope},
		TenantID: auth.cfg.TenantID,
	}

	token, err := cred.GetToken(context.Background(), opts)
	if err != nil {
		return "", fmt.Errorf("failed to acquire Entra ID token: %w", err)
	}

	return token.Token, nil
}

func (auth *EntraIDAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *EntraIDAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *EntraIDAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

type Claims struct {
	TenantID string `json:"tid,omitempty"`
	jwt.RegisteredClaims
}

type EntraIDAuthVerifier struct {
	additionalAuthScopes []v1.AuthScope
	cfg                  v1.AuthEntraIDServerConfig
	jwks                 keyfunc.Keyfunc
}

func NewEntraIDAuthVerifier(additionalAuthScopes []v1.AuthScope, cfg v1.AuthEntraIDServerConfig) (*EntraIDAuthVerifier, error) {
	jwksURL := "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Microsoft JWK Set: %w", err)
	}

	return &EntraIDAuthVerifier{
		additionalAuthScopes: additionalAuthScopes,
		cfg:                  cfg,
		jwks:                 jwks,
	}, nil
}

func (auth *EntraIDAuthVerifier) verifyToken(token string) error {
	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, auth.jwks.Keyfunc)
	if err != nil {
		return fmt.Errorf("error parsing or validating token: %w", err)
	}
	if !parsedToken.Valid {
		return fmt.Errorf("token is invalid: failed internal validation checks")
	}

	// Validate Audience - check if audience is in the slice
	if !slices.Contains(claims.Audience, auth.cfg.Audience) {
		return fmt.Errorf("invalid audience: expected %s, but token contains %v", auth.cfg.Audience, claims.Audience)
	}

	// Validate Tenant ID (if configured)
	if auth.cfg.TenantID != "" && claims.TenantID != auth.cfg.TenantID {
		return fmt.Errorf("invalid tenant ID: expected %s, but got %s", auth.cfg.TenantID, claims.TenantID)
	}

	// Dynamic Issuer validation - support both v1.0 and v2.0 formats
	expectedIssuerV1 := fmt.Sprintf("https://sts.windows.net/%s/", claims.TenantID)
	expectedIssuerV2 := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", claims.TenantID)
	if claims.Issuer != expectedIssuerV1 && claims.Issuer != expectedIssuerV2 {
		return fmt.Errorf("invalid issuer: expected %s or %s, but got %s", expectedIssuerV1, expectedIssuerV2, claims.Issuer)
	}

	return nil
}

func (auth *EntraIDAuthVerifier) VerifyLogin(m *msg.Login) error {
	return auth.verifyToken(m.PrivilegeKey)
}

func (auth *EntraIDAuthVerifier) VerifyPing(m *msg.Ping) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}
	return auth.verifyToken(m.PrivilegeKey)
}

func (auth *EntraIDAuthVerifier) VerifyNewWorkConn(m *msg.NewWorkConn) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}
	return auth.verifyToken(m.PrivilegeKey)
}
