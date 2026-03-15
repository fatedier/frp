package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
)

type mockTokenVerifier struct{}

func (m *mockTokenVerifier) Verify(ctx context.Context, subject string) (*oidc.IDToken, error) {
	return &oidc.IDToken{
		Subject: subject,
	}, nil
}

func TestPingWithEmptySubjectFromLoginFails(t *testing.T) {
	r := require.New(t)
	consumer := auth.NewOidcAuthVerifier([]v1.AuthScope{v1.AuthScopeHeartBeats}, &mockTokenVerifier{})
	err := consumer.VerifyPing(&msg.Ping{
		PrivilegeKey: "ping-without-login",
		Timestamp:    time.Now().UnixMilli(),
	})
	r.Error(err)
	r.Contains(err.Error(), "received different OIDC subject in login and ping")
}

func TestPingAfterLoginWithNewSubjectSucceeds(t *testing.T) {
	r := require.New(t)
	consumer := auth.NewOidcAuthVerifier([]v1.AuthScope{v1.AuthScopeHeartBeats}, &mockTokenVerifier{})
	err := consumer.VerifyLogin(&msg.Login{
		PrivilegeKey: "ping-after-login",
	})
	r.NoError(err)

	err = consumer.VerifyPing(&msg.Ping{
		PrivilegeKey: "ping-after-login",
		Timestamp:    time.Now().UnixMilli(),
	})
	r.NoError(err)
}

func TestPingAfterLoginWithDifferentSubjectFails(t *testing.T) {
	r := require.New(t)
	consumer := auth.NewOidcAuthVerifier([]v1.AuthScope{v1.AuthScopeHeartBeats}, &mockTokenVerifier{})
	err := consumer.VerifyLogin(&msg.Login{
		PrivilegeKey: "login-with-first-subject",
	})
	r.NoError(err)

	err = consumer.VerifyPing(&msg.Ping{
		PrivilegeKey: "ping-with-different-subject",
		Timestamp:    time.Now().UnixMilli(),
	})
	r.Error(err)
	r.Contains(err.Error(), "received different OIDC subject in login and ping")
}

func TestOidcAuthProviderFallsBackWhenNoExpiry(t *testing.T) {
	r := require.New(t)

	var requestCount atomic.Int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:gosec // test-only dummy token response
			"access_token": "fresh-test-token",
			"token_type":   "Bearer",
		})
	}))
	defer tokenServer.Close()

	provider, err := auth.NewOidcAuthSetter(
		[]v1.AuthScope{v1.AuthScopeHeartBeats},
		v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			ClientSecret:     "test-secret",
			TokenEndpointURL: tokenServer.URL,
		},
	)
	r.NoError(err)

	// Constructor no longer fetches a token eagerly.
	// The first SetLogin triggers the adaptive probe.
	r.Equal(int32(0), requestCount.Load())

	loginMsg := &msg.Login{}
	err = provider.SetLogin(loginMsg)
	r.NoError(err)
	r.Equal("fresh-test-token", loginMsg.PrivilegeKey)

	for range 3 {
		pingMsg := &msg.Ping{}
		err = provider.SetPing(pingMsg)
		r.NoError(err)
		r.Equal("fresh-test-token", pingMsg.PrivilegeKey)
	}

	// 1 probe (login) + 3 pings = 4 requests (probe doubles as the login token fetch)
	r.Equal(int32(4), requestCount.Load(), "each call should fetch a fresh token when expires_in is missing")
}

func TestOidcAuthProviderCachesToken(t *testing.T) {
	r := require.New(t)

	var requestCount atomic.Int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:gosec // test-only dummy token response
			"access_token": "cached-test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	provider, err := auth.NewOidcAuthSetter(
		[]v1.AuthScope{v1.AuthScopeHeartBeats},
		v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			ClientSecret:     "test-secret",
			TokenEndpointURL: tokenServer.URL,
		},
	)
	r.NoError(err)

	// Constructor no longer fetches eagerly; first SetLogin triggers the probe.
	r.Equal(int32(0), requestCount.Load())

	// SetLogin triggers the adaptive probe and caches the token.
	loginMsg := &msg.Login{}
	err = provider.SetLogin(loginMsg)
	r.NoError(err)
	r.Equal("cached-test-token", loginMsg.PrivilegeKey)
	r.Equal(int32(1), requestCount.Load())

	// Subsequent calls should also reuse the cached token
	for range 5 {
		pingMsg := &msg.Ping{}
		err = provider.SetPing(pingMsg)
		r.NoError(err)
		r.Equal("cached-test-token", pingMsg.PrivilegeKey)
	}
	r.Equal(int32(1), requestCount.Load(), "token endpoint should only be called once; cached token should be reused")
}

func TestOidcAuthProviderRetriesOnInitialFailure(t *testing.T) {
	r := require.New(t)

	var requestCount atomic.Int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		n := requestCount.Add(1)
		// The oauth2 library retries once internally, so we need two
		// consecutive failures to surface an error to the caller.
		if n <= 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":             "temporarily_unavailable",
				"error_description": "service is starting up",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:gosec // test-only dummy token response
			"access_token": "retry-test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	// Constructor succeeds even though the IdP is "down".
	provider, err := auth.NewOidcAuthSetter(
		[]v1.AuthScope{v1.AuthScopeHeartBeats},
		v1.AuthOIDCClientConfig{
			ClientID:         "test-client",
			ClientSecret:     "test-secret",
			TokenEndpointURL: tokenServer.URL,
		},
	)
	r.NoError(err)
	r.Equal(int32(0), requestCount.Load())

	// First SetLogin hits the IdP, which returns an error (after internal retry).
	loginMsg := &msg.Login{}
	err = provider.SetLogin(loginMsg)
	r.Error(err)
	r.Equal(int32(2), requestCount.Load())

	// Second SetLogin retries and succeeds.
	err = provider.SetLogin(loginMsg)
	r.NoError(err)
	r.Equal("retry-test-token", loginMsg.PrivilegeKey)
	r.Equal(int32(3), requestCount.Load())

	// Subsequent calls use cached token.
	pingMsg := &msg.Ping{}
	err = provider.SetPing(pingMsg)
	r.NoError(err)
	r.Equal("retry-test-token", pingMsg.PrivilegeKey)
	r.Equal(int32(3), requestCount.Load())
}

func TestNewOidcAuthSetterRejectsInvalidStaticConfig(t *testing.T) {
	r := require.New(t)
	tokenServer := httptest.NewServer(http.NotFoundHandler())
	defer tokenServer.Close()

	_, err := auth.NewOidcAuthSetter(nil, v1.AuthOIDCClientConfig{
		ClientID:         "test-client",
		TokenEndpointURL: "://bad",
	})
	r.Error(err)
	r.Contains(err.Error(), "auth.oidc.tokenEndpointURL")

	_, err = auth.NewOidcAuthSetter(nil, v1.AuthOIDCClientConfig{
		TokenEndpointURL: tokenServer.URL,
	})
	r.Error(err)
	r.Contains(err.Error(), "auth.oidc.clientID is required")

	_, err = auth.NewOidcAuthSetter(nil, v1.AuthOIDCClientConfig{
		ClientID:         "test-client",
		TokenEndpointURL: tokenServer.URL,
		AdditionalEndpointParams: map[string]string{
			"scope": "profile",
		},
	})
	r.Error(err)
	r.Contains(err.Error(), "auth.oidc.additionalEndpointParams.scope is not allowed; use auth.oidc.scope instead")

	_, err = auth.NewOidcAuthSetter(nil, v1.AuthOIDCClientConfig{
		ClientID:                 "test-client",
		TokenEndpointURL:         tokenServer.URL,
		Audience:                 "api",
		AdditionalEndpointParams: map[string]string{"audience": "override"},
	})
	r.Error(err)
	r.Contains(err.Error(), "cannot specify both auth.oidc.audience and auth.oidc.additionalEndpointParams.audience")
}
