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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:gosec // test-only dummy token response
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

	// Constructor fetches the initial token (1 request).
	// Each subsequent call should also fetch a fresh token since there is no expiry.
	loginMsg := &msg.Login{}
	err = provider.SetLogin(loginMsg)
	r.NoError(err)
	r.Equal("fresh-test-token", loginMsg.PrivilegeKey)

	for i := 0; i < 3; i++ {
		pingMsg := &msg.Ping{}
		err = provider.SetPing(pingMsg)
		r.NoError(err)
		r.Equal("fresh-test-token", pingMsg.PrivilegeKey)
	}

	// 1 initial (constructor) + 1 login + 3 pings = 5 requests
	r.Equal(int32(5), requestCount.Load(), "each call should fetch a fresh token when expires_in is missing")
}

func TestOidcAuthProviderCachesToken(t *testing.T) {
	r := require.New(t)

	var requestCount atomic.Int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:gosec // test-only dummy token response
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

	// Constructor eagerly fetches the initial token (1 request).
	r.Equal(int32(1), requestCount.Load())

	// SetLogin should reuse the cached token
	loginMsg := &msg.Login{}
	err = provider.SetLogin(loginMsg)
	r.NoError(err)
	r.Equal("cached-test-token", loginMsg.PrivilegeKey)
	r.Equal(int32(1), requestCount.Load())

	// Subsequent calls should also reuse the cached token
	for i := 0; i < 5; i++ {
		pingMsg := &msg.Ping{}
		err = provider.SetPing(pingMsg)
		r.NoError(err)
		r.Equal("cached-test-token", pingMsg.PrivilegeKey)
	}
	r.Equal(int32(1), requestCount.Load(), "token endpoint should only be called once; cached token should be reused")
}
