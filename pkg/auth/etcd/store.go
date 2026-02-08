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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/util/log"
)

const (
	defaultPrefix      = "/frp/tokens/"
	defaultDialTimeout = 5
)

// TokenInvalidCallback is called when a token is deleted or disabled.
type TokenInvalidCallback func(token string)

// TokenStore manages token configurations from etcd.
type TokenStore struct {
	client *clientv3.Client
	cfg    *v1.EtcdConfig

	// Cache for token configurations
	cache map[string]*v1.TokenConfig
	mu    sync.RWMutex

	// Callback when token is invalidated (deleted or disabled)
	onTokenInvalid TokenInvalidCallback

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTokenStore creates a new TokenStore connected to etcd.
func NewTokenStore(cfg *v1.EtcdConfig) (*TokenStore, error) {
	if !cfg.IsEnabled() {
		return nil, fmt.Errorf("etcd configuration is not enabled")
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = defaultPrefix
	}
	cfg.Prefix = prefix

	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = defaultDialTimeout
	}

	etcdCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: time.Duration(dialTimeout) * time.Second,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}

	// Configure TLS if provided
	if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load etcd TLS cert: %w", err)
		}
		etcdCfg.TLS = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	client, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	store := &TokenStore{
		client: client,
		cfg:    cfg,
		cache:  make(map[string]*v1.TokenConfig),
		ctx:    ctx,
		cancel: cancel,
	}

	// Initial load of all tokens
	if err := store.loadAllTokens(); err != nil {
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to load tokens from etcd: %w", err)
	}

	// Start watching for changes
	go store.watchTokens()

	log.Infof("etcd token store initialized with %d tokens for region [%s]", len(store.cache), cfg.Region)
	return store, nil
}

// GetToken retrieves a token configuration by token string.
// Returns nil if token not found or not valid for this region.
func (s *TokenStore) GetToken(token string) *v1.TokenConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tc, ok := s.cache[token]
	if !ok {
		return nil
	}

	// Verify region matches
	if tc.Region != s.cfg.Region {
		return nil
	}

	// Verify token is enabled
	if !tc.Enabled {
		return nil
	}

	return tc
}

// ValidateToken checks if a token is valid for this region.
func (s *TokenStore) ValidateToken(token string) (*v1.TokenConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tc, ok := s.cache[token]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}

	if tc.Region != s.cfg.Region {
		return nil, fmt.Errorf("token region [%s] does not match server region [%s]", tc.Region, s.cfg.Region)
	}

	if !tc.Enabled {
		return nil, fmt.Errorf("token is disabled")
	}

	return tc, nil
}

// loadAllTokens loads all tokens from etcd.
func (s *TokenStore) loadAllTokens() error {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	resp, err := s.client.Get(ctx, s.cfg.Prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing cache
	s.cache = make(map[string]*v1.TokenConfig)

	for _, kv := range resp.Kvs {
		var tc v1.TokenConfig
		if err := json.Unmarshal(kv.Value, &tc); err != nil {
			log.Warnf("failed to parse token config from etcd key [%s]: %v", string(kv.Key), err)
			continue
		}
		s.cache[tc.Token] = &tc
		log.Debugf("loaded token [%s] for region [%s]", tc.Token[:min(8, len(tc.Token))]+"...", tc.Region)
	}

	return nil
}

// watchTokens watches for changes in etcd and updates the cache.
func (s *TokenStore) watchTokens() {
	watchChan := s.client.Watch(s.ctx, s.cfg.Prefix, clientv3.WithPrefix())

	for {
		select {
		case <-s.ctx.Done():
			return
		case resp, ok := <-watchChan:
			if !ok {
				log.Warnf("etcd watch channel closed, attempting to reconnect...")
				time.Sleep(time.Second)
				watchChan = s.client.Watch(s.ctx, s.cfg.Prefix, clientv3.WithPrefix())
				continue
			}

			for _, event := range resp.Events {
				s.handleWatchEvent(event)
			}
		}
	}
}

// handleWatchEvent processes a single watch event.
func (s *TokenStore) handleWatchEvent(event *clientv3.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case clientv3.EventTypePut:
		var tc v1.TokenConfig
		if err := json.Unmarshal(event.Kv.Value, &tc); err != nil {
			log.Warnf("failed to parse token config from etcd key [%s]: %v", string(event.Kv.Key), err)
			return
		}

		// Check if token was previously enabled and is now disabled
		oldTc, existed := s.cache[tc.Token]
		wasEnabled := existed && oldTc.Enabled && oldTc.Region == s.cfg.Region

		s.cache[tc.Token] = &tc
		log.Infof("token [%s] updated/added for region [%s]", tc.Token[:min(8, len(tc.Token))]+"...", tc.Region)

		// If token is now disabled or region changed, trigger callback
		isValidNow := tc.Enabled && tc.Region == s.cfg.Region
		if wasEnabled && !isValidNow && s.onTokenInvalid != nil {
			log.Infof("token [%s] invalidated (disabled or region changed), closing connections", tc.Token[:min(8, len(tc.Token))]+"...")
			go s.onTokenInvalid(tc.Token)
		}

	case clientv3.EventTypeDelete:
		// Extract token from key
		key := string(event.Kv.Key)
		// Find and remove the token from cache
		for token, tc := range s.cache {
			if key == s.cfg.Prefix+token {
				wasValidForRegion := tc.Enabled && tc.Region == s.cfg.Region
				delete(s.cache, token)
				log.Infof("token [%s] deleted", token[:min(8, len(token))]+"...")

				// Trigger callback to close connections using this token
				if wasValidForRegion && s.onTokenInvalid != nil {
					log.Infof("token [%s] deleted, closing connections", token[:min(8, len(token))]+"...")
					go s.onTokenInvalid(token)
				}
				break
			}
		}
	}
}

// Close closes the etcd connection.
func (s *TokenStore) Close() error {
	s.cancel()
	return s.client.Close()
}

// SetTokenInvalidCallback sets the callback function that will be called
// when a token is deleted or disabled.
func (s *TokenStore) SetTokenInvalidCallback(cb TokenInvalidCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTokenInvalid = cb
}

// Region returns the configured region.
func (s *TokenStore) Region() string {
	return s.cfg.Region
}

// TokenCount returns the number of cached tokens.
func (s *TokenStore) TokenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}
