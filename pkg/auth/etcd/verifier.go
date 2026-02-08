// Copyright 2024 The frp Authors
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

package etcd

import (
	"fmt"
	"slices"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

// MultiTokenVerifier implements auth.Verifier interface for etcd-based multi-token authentication.
type MultiTokenVerifier struct {
	store            *TokenStore
	additionalScopes []v1.AuthScope
}

// NewMultiTokenVerifier creates a new MultiTokenVerifier.
func NewMultiTokenVerifier(store *TokenStore, additionalScopes []v1.AuthScope) *MultiTokenVerifier {
	return &MultiTokenVerifier{
		store:            store,
		additionalScopes: additionalScopes,
	}
}

// VerifyLogin verifies the login message against tokens stored in etcd.
// It tries to find a matching token and validates the region.
func (v *MultiTokenVerifier) VerifyLogin(m *msg.Login) error {
	// Try each token in the store to find a match
	token, err := v.findMatchingToken(m.PrivilegeKey, m.Timestamp)
	if err != nil {
		return err
	}

	// Validate the token configuration
	tc, err := v.store.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Store token config in login metadata for later use
	if m.Metas == nil {
		m.Metas = make(map[string]string)
	}
	m.Metas["_etcd_token"] = tc.Token

	return nil
}

// VerifyPing verifies the ping message.
func (v *MultiTokenVerifier) VerifyPing(m *msg.Ping) error {
	if !slices.Contains(v.additionalScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	_, err := v.findMatchingToken(m.PrivilegeKey, m.Timestamp)
	if err != nil {
		return fmt.Errorf("token in heartbeat doesn't match any valid token")
	}
	return nil
}

// VerifyNewWorkConn verifies the new work connection message.
func (v *MultiTokenVerifier) VerifyNewWorkConn(m *msg.NewWorkConn) error {
	if !slices.Contains(v.additionalScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	_, err := v.findMatchingToken(m.PrivilegeKey, m.Timestamp)
	if err != nil {
		return fmt.Errorf("token in NewWorkConn doesn't match any valid token")
	}
	return nil
}

// findMatchingToken finds the token that matches the given privilege key.
func (v *MultiTokenVerifier) findMatchingToken(privilegeKey string, timestamp int64) (string, error) {
	v.store.mu.RLock()
	defer v.store.mu.RUnlock()

	for token, tc := range v.store.cache {
		// Skip disabled tokens
		if !tc.Enabled {
			continue
		}

		// Skip tokens for other regions
		if tc.Region != v.store.cfg.Region {
			continue
		}

		// Calculate expected key for this token
		expectedKey := util.GetAuthKey(token, timestamp)
		if util.ConstantTimeEqString(expectedKey, privilegeKey) {
			return token, nil
		}
	}

	return "", fmt.Errorf("no matching token found")
}

// GetTokenConfig returns the token configuration for a given token.
func (v *MultiTokenVerifier) GetTokenConfig(token string) *v1.TokenConfig {
	return v.store.GetToken(token)
}
