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

// Package oidcserver provides a minimal mock OIDC server for e2e testing.
// It implements three endpoints:
//   - /.well-known/openid-configuration (discovery)
//   - /jwks (JSON Web Key Set)
//   - /token (client_credentials grant)
package oidcserver

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type Server struct {
	bindAddr string
	bindPort int
	l        net.Listener
	hs       *http.Server

	privateKey *rsa.PrivateKey
	kid        string

	clientID     string
	clientSecret string
	audience     string
	subject      string
	expiresIn    int // seconds; 0 means omit expires_in from token response

	tokenRequestCount atomic.Int64
}

type Option func(*Server)

func WithBindPort(port int) Option {
	return func(s *Server) { s.bindPort = port }
}

func WithClientCredentials(id, secret string) Option {
	return func(s *Server) {
		s.clientID = id
		s.clientSecret = secret
	}
}

func WithAudience(aud string) Option {
	return func(s *Server) { s.audience = aud }
}

func WithSubject(sub string) Option {
	return func(s *Server) { s.subject = sub }
}

func WithExpiresIn(seconds int) Option {
	return func(s *Server) { s.expiresIn = seconds }
}

func New(options ...Option) *Server {
	s := &Server{
		bindAddr:     "127.0.0.1",
		kid:          "test-key-1",
		clientID:     "test-client",
		clientSecret: "test-secret",
		audience:     "frps",
		subject:      "test-service",
		expiresIn:    3600,
	}
	for _, opt := range options {
		opt(s)
	}
	return s
}

func (s *Server) Run() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate RSA key: %w", err)
	}
	s.privateKey = key

	s.l, err = net.Listen("tcp", net.JoinHostPort(s.bindAddr, strconv.Itoa(s.bindPort)))
	if err != nil {
		return err
	}
	s.bindPort = s.l.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", s.handleDiscovery)
	mux.HandleFunc("/jwks", s.handleJWKS)
	mux.HandleFunc("/token", s.handleToken)

	s.hs = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Minute,
	}
	go func() { _ = s.hs.Serve(s.l) }()
	return nil
}

func (s *Server) Close() error {
	if s.hs != nil {
		return s.hs.Close()
	}
	return nil
}

func (s *Server) BindAddr() string { return s.bindAddr }
func (s *Server) BindPort() int    { return s.bindPort }

func (s *Server) Issuer() string {
	return fmt.Sprintf("http://%s:%d", s.bindAddr, s.bindPort)
}

func (s *Server) TokenEndpoint() string {
	return s.Issuer() + "/token"
}

// TokenRequestCount returns the number of successful token requests served.
func (s *Server) TokenRequestCount() int64 {
	return s.tokenRequestCount.Load()
}

func (s *Server) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	issuer := s.Issuer()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                                issuer,
		"token_endpoint":                        issuer + "/token",
		"jwks_uri":                              issuer + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (s *Server) handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pub := &s.privateKey.PublicKey
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"kid": s.kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	})
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_request",
		})
		return
	}

	if r.FormValue("grant_type") != "client_credentials" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "unsupported_grant_type",
		})
		return
	}

	// Accept credentials from Basic Auth or form body.
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if clientID != s.clientID || clientSecret != s.clientSecret {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_client",
		})
		return
	}

	token, err := s.signJWT()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
	}
	if s.expiresIn > 0 {
		resp["expires_in"] = s.expiresIn
	}

	s.tokenRequestCount.Add(1)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) signJWT() (string, error) {
	now := time.Now()
	header, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"kid": s.kid,
		"typ": "JWT",
	})
	claims, _ := json.Marshal(map[string]any{
		"iss": s.Issuer(),
		"sub": s.subject,
		"aud": s.audience,
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
	})

	headerB64 := base64.RawURLEncoding.EncodeToString(header)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claims)
	signingInput := headerB64 + "." + claimsB64

	h := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
