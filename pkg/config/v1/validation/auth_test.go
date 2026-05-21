// Copyright 2026 The frp Authors
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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/policy/security"
)

const (
	tokenSourceConflictErr = "cannot specify both auth.token and auth.tokenSource"
	tokenSourceExecErr     = "unsafe feature \"TokenSourceExec\" is not enabled. To enable it, ensure it is allowed in the configuration or command line flags"
	invalidFileSourceErr   = "invalid auth.tokenSource: file configuration is required when type is 'file'"
	unsupportedSourceErr   = "invalid auth.tokenSource: unsupported value source type: env (only 'file' and 'exec' are supported)"
)

func TestValidateAuthTokenSource(t *testing.T) {
	for _, tc := range authTokenSourceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			validator := newAuthTokenSourceValidator(tc.unsafeAllowed)
			err := validator.validateAuthTokenSource(tc.token, tc.tokenSource())
			requireValidationErrors(t, err, tc.wantErrs)
		})
	}
}

func TestValidateClientAuthTokenSource(t *testing.T) {
	for _, tc := range authTokenSourceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			auth := v1.AuthClientConfig{
				Method:      v1.AuthMethodToken,
				Token:       tc.token,
				TokenSource: tc.tokenSource(),
			}
			validator := newAuthTokenSourceValidator(tc.unsafeAllowed)
			_, err := validator.ValidateClientCommonConfig(validClientConfigWithAuth(auth))
			requireValidationErrors(t, err, tc.wantErrs)
		})
	}
}

func TestValidateServerAuthTokenSource(t *testing.T) {
	for _, tc := range authTokenSourceTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			auth := v1.AuthServerConfig{
				Method:      v1.AuthMethodToken,
				Token:       tc.token,
				TokenSource: tc.tokenSource(),
			}
			validator := newAuthTokenSourceValidator(tc.unsafeAllowed)
			_, err := validator.ValidateServerConfig(validServerConfigWithAuth(auth))
			requireValidationErrors(t, err, tc.wantErrs)
		})
	}
}

type authTokenSourceTestCase struct {
	name          string
	token         string
	tokenSource   func() *v1.ValueSource
	unsafeAllowed bool
	wantErrs      []string
}

func authTokenSourceTestCases() []authTokenSourceTestCase {
	return []authTokenSourceTestCase{
		{
			name:        "empty token config",
			tokenSource: nilTokenSource,
		},
		{
			name:        "valid file tokenSource",
			tokenSource: validFileTokenSource,
		},
		{
			name:        "literal token without tokenSource",
			token:       "token",
			tokenSource: nilTokenSource,
		},
		{
			name:        "literal token conflicts with file tokenSource",
			token:       "token",
			tokenSource: validFileTokenSource,
			wantErrs:    []string{tokenSourceConflictErr},
		},
		{
			name:        "exec tokenSource requires unsafe feature",
			tokenSource: validExecTokenSource,
			wantErrs:    []string{tokenSourceExecErr},
		},
		{
			name:          "exec tokenSource with unsafe feature allowed",
			tokenSource:   validExecTokenSource,
			unsafeAllowed: true,
		},
		{
			name:        "literal token conflicts with exec tokenSource and unsafe feature disabled",
			token:       "token",
			tokenSource: validExecTokenSource,
			wantErrs: []string{
				tokenSourceConflictErr,
				tokenSourceExecErr,
			},
		},
		{
			name:          "literal token conflicts with exec tokenSource and unsafe feature allowed",
			token:         "token",
			tokenSource:   validExecTokenSource,
			unsafeAllowed: true,
			wantErrs:      []string{tokenSourceConflictErr},
		},
		{
			name:        "invalid file tokenSource is wrapped",
			tokenSource: invalidFileTokenSource,
			wantErrs:    []string{invalidFileSourceErr},
		},
		{
			name:        "unsupported tokenSource type is wrapped",
			tokenSource: unsupportedTokenSource,
			wantErrs:    []string{unsupportedSourceErr},
		},
	}
}

func newAuthTokenSourceValidator(unsafeAllowed bool) *ConfigValidator {
	if !unsafeAllowed {
		return NewConfigValidator(nil)
	}
	return NewConfigValidator(security.NewUnsafeFeatures([]string{security.TokenSourceExec}))
}

func requireValidationErrors(t *testing.T, err error, wantErrs []string) {
	t.Helper()
	if len(wantErrs) == 0 {
		require.NoError(t, err)
		return
	}
	require.Error(t, err)
	// Client/server validators may wrap joined errors in another join layer; compare leaf errors.
	gotErrs := unwrapValidationErrors(err)
	require.Len(t, gotErrs, len(wantErrs))
	for i, wantErr := range wantErrs {
		require.EqualError(t, gotErrs[i], wantErr)
	}
}

func unwrapValidationErrors(err error) []error {
	type joinedError interface {
		Unwrap() []error
	}
	joined, ok := err.(joinedError)
	if !ok {
		return []error{err}
	}

	var errs []error
	for _, err := range joined.Unwrap() {
		errs = append(errs, unwrapValidationErrors(err)...)
	}
	return errs
}

// nilTokenSource keeps the shared table shape uniform for cases without a tokenSource.
func nilTokenSource() *v1.ValueSource {
	return nil
}

func validFileTokenSource() *v1.ValueSource {
	return &v1.ValueSource{
		Type: "file",
		File: &v1.FileSource{Path: "token.txt"},
	}
}

func validExecTokenSource() *v1.ValueSource {
	return &v1.ValueSource{
		Type: "exec",
		Exec: &v1.ExecSource{Command: "print-token"},
	}
}

func invalidFileTokenSource() *v1.ValueSource {
	return &v1.ValueSource{
		Type: "file",
	}
}

func unsupportedTokenSource() *v1.ValueSource {
	return &v1.ValueSource{Type: "env"}
}

func validClientConfigWithAuth(auth v1.AuthClientConfig) *v1.ClientCommonConfig {
	return &v1.ClientCommonConfig{
		Auth: auth,
		Log: v1.LogConfig{
			Level: "info",
		},
		Transport: v1.ClientTransportConfig{
			Protocol:     "tcp",
			WireProtocol: "v1",
		},
	}
}

func validServerConfigWithAuth(auth v1.AuthServerConfig) *v1.ServerConfig {
	return &v1.ServerConfig{
		Auth: auth,
		Log: v1.LogConfig{
			Level: "info",
		},
	}
}
