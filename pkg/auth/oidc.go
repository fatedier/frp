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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
)

// createOIDCHTTPClient creates an HTTP client with custom TLS and proxy configuration for OIDC token requests
func createOIDCHTTPClient(trustedCAFile string, insecureSkipVerify bool, proxyURL string) (*http.Client, error) {
	// Clone the default transport to get all reasonable defaults
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Configure TLS settings
	if trustedCAFile != "" || insecureSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		}

		if trustedCAFile != "" && !insecureSkipVerify {
			caCert, err := os.ReadFile(trustedCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read OIDC CA certificate file %q: %w", trustedCAFile, err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse OIDC CA certificate from file %q", trustedCAFile)
			}

			tlsConfig.RootCAs = caCertPool
		}
		transport.TLSClientConfig = tlsConfig
	}

	// Configure proxy settings
	if proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OIDC proxy URL %q: %w", proxyURL, err)
		}
		transport.Proxy = http.ProxyURL(parsedURL)
	} else {
		// Explicitly disable proxy to override DefaultTransport's ProxyFromEnvironment
		transport.Proxy = nil
	}

	return &http.Client{Transport: transport}, nil
}

// nonCachingTokenSource wraps a clientcredentials.Config to fetch a fresh
// token on every call. This is used as a fallback when the OIDC provider
// does not return expires_in, which would cause a caching TokenSource to
// hold onto a stale token forever.
type nonCachingTokenSource struct {
	cfg *clientcredentials.Config
	ctx context.Context
}

func (s *nonCachingTokenSource) Token() (*oauth2.Token, error) {
	return s.cfg.Token(s.ctx)
}

// oidcTokenSource wraps a caching oauth2.TokenSource and, on the first
// successful Token() call, checks whether the provider returns an expiry.
// If not, it permanently switches to nonCachingTokenSource so that a fresh
// token is fetched every time.  This avoids an eager network call at
// construction time, letting the login retry loop handle transient IdP
// outages.
type oidcTokenSource struct {
	mu          sync.Mutex
	initialized bool
	source      oauth2.TokenSource
	fallbackCfg *clientcredentials.Config
	fallbackCtx context.Context
}

func (s *oidcTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	if !s.initialized {
		token, err := s.source.Token()
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		if token.Expiry.IsZero() {
			s.source = &nonCachingTokenSource{cfg: s.fallbackCfg, ctx: s.fallbackCtx}
		}
		s.initialized = true
		s.mu.Unlock()
		return token, nil
	}
	source := s.source
	s.mu.Unlock()
	return source.Token()
}

type OidcAuthProvider struct {
	additionalAuthScopes []v1.AuthScope

	tokenSource oauth2.TokenSource
}

func NewOidcAuthSetter(additionalAuthScopes []v1.AuthScope, cfg v1.AuthOIDCClientConfig) (*OidcAuthProvider, error) {
	if err := validation.ValidateOIDCClientCredentialsConfig(&cfg); err != nil {
		return nil, err
	}

	eps := make(map[string][]string)
	for k, v := range cfg.AdditionalEndpointParams {
		eps[k] = []string{v}
	}

	if cfg.Audience != "" {
		eps["audience"] = []string{cfg.Audience}
	}

	tokenGenerator := &clientcredentials.Config{
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		Scopes:         []string{cfg.Scope},
		TokenURL:       cfg.TokenEndpointURL,
		EndpointParams: eps,
	}

	// Build the context that TokenSource will use for all future HTTP requests.
	// context.Background() is appropriate here because the token source is
	// long-lived and outlives any single request.
	ctx := context.Background()
	if cfg.TrustedCaFile != "" || cfg.InsecureSkipVerify || cfg.ProxyURL != "" {
		httpClient, err := createOIDCHTTPClient(cfg.TrustedCaFile, cfg.InsecureSkipVerify, cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC HTTP client: %w", err)
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	// Create a persistent TokenSource that caches the token and refreshes
	// it before expiry. This avoids making a new HTTP request to the OIDC
	// provider on every heartbeat/ping.
	//
	// We wrap it in an oidcTokenSource so that the first Token() call
	// (deferred to SetLogin inside the login retry loop) probes whether the
	// provider returns expires_in.  If not, it switches to a non-caching
	// source.  This avoids an eager network call at construction time, which
	// would prevent loopLoginUntilSuccess from retrying on transient IdP
	// outages.
	cachingSource := tokenGenerator.TokenSource(ctx)

	return &OidcAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		tokenSource: &oidcTokenSource{
			source:      cachingSource,
			fallbackCfg: tokenGenerator,
			fallbackCtx: ctx,
		},
	}, nil
}

func (auth *OidcAuthProvider) generateAccessToken() (accessToken string, err error) {
	tokenObj, err := auth.tokenSource.Token()
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
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

type OidcTokenSourceAuthProvider struct {
	additionalAuthScopes []v1.AuthScope

	valueSource *v1.ValueSource
}

func NewOidcTokenSourceAuthSetter(additionalAuthScopes []v1.AuthScope, valueSource *v1.ValueSource) *OidcTokenSourceAuthProvider {
	return &OidcTokenSourceAuthProvider{
		additionalAuthScopes: additionalAuthScopes,
		valueSource:          valueSource,
	}
}

func (auth *OidcTokenSourceAuthProvider) generateAccessToken() (accessToken string, err error) {
	ctx := context.Background()
	accessToken, err = auth.valueSource.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("couldn't acquire OIDC token for login: %v", err)
	}
	return
}

func (auth *OidcTokenSourceAuthProvider) SetLogin(loginMsg *msg.Login) (err error) {
	loginMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcTokenSourceAuthProvider) SetPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

func (auth *OidcTokenSourceAuthProvider) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.PrivilegeKey, err = auth.generateAccessToken()
	return err
}

type TokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type OidcAuthConsumer struct {
	additionalAuthScopes []v1.AuthScope

	verifier          TokenVerifier
	mu                sync.RWMutex
	subjectsFromLogin map[string]struct{}
}

func NewTokenVerifier(cfg v1.AuthOIDCServerConfig) TokenVerifier {
	provider, err := oidc.NewProvider(context.Background(), cfg.Issuer)
	if err != nil {
		panic(err)
	}
	verifierConf := oidc.Config{
		ClientID:          cfg.Audience,
		SkipClientIDCheck: cfg.Audience == "",
		SkipExpiryCheck:   cfg.SkipExpiryCheck,
		SkipIssuerCheck:   cfg.SkipIssuerCheck,
	}
	return provider.Verifier(&verifierConf)
}

func NewOidcAuthVerifier(additionalAuthScopes []v1.AuthScope, verifier TokenVerifier) *OidcAuthConsumer {
	return &OidcAuthConsumer{
		additionalAuthScopes: additionalAuthScopes,
		verifier:             verifier,
		subjectsFromLogin:    make(map[string]struct{}),
	}
}

func (auth *OidcAuthConsumer) VerifyLogin(loginMsg *msg.Login) (err error) {
	token, err := auth.verifier.Verify(context.Background(), loginMsg.PrivilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in login: %v", err)
	}
	auth.mu.Lock()
	auth.subjectsFromLogin[token.Subject] = struct{}{}
	auth.mu.Unlock()
	return nil
}

func (auth *OidcAuthConsumer) verifyPostLoginToken(privilegeKey string) (err error) {
	token, err := auth.verifier.Verify(context.Background(), privilegeKey)
	if err != nil {
		return fmt.Errorf("invalid OIDC token in ping: %v", err)
	}
	auth.mu.RLock()
	_, ok := auth.subjectsFromLogin[token.Subject]
	auth.mu.RUnlock()
	if !ok {
		return fmt.Errorf("received different OIDC subject in login and ping. "+
			"new subject: %s",
			token.Subject)
	}
	return nil
}

func (auth *OidcAuthConsumer) VerifyPing(pingMsg *msg.Ping) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	return auth.verifyPostLoginToken(pingMsg.PrivilegeKey)
}

func (auth *OidcAuthConsumer) VerifyNewWorkConn(newWorkConnMsg *msg.NewWorkConn) (err error) {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	return auth.verifyPostLoginToken(newWorkConnMsg.PrivilegeKey)
}
