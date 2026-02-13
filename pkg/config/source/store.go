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

package source

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type StoreSourceConfig struct {
	Path string `json:"path"`
}

type storeData struct {
	Proxies  []v1.TypedProxyConfig   `json:"proxies,omitempty"`
	Visitors []v1.TypedVisitorConfig `json:"visitors,omitempty"`
}

type StoreSource struct {
	baseSource
	config StoreSourceConfig
}

func NewStoreSource(cfg StoreSourceConfig) (*StoreSource, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	s := &StoreSource{
		baseSource: newBaseSource(),
		config:     cfg,
	}

	if err := s.loadFromFile(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load existing data: %w", err)
		}
	}

	return s, nil
}

func (s *StoreSource) loadFromFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadFromFileUnlocked()
}

func (s *StoreSource) loadFromFileUnlocked() error {
	data, err := os.ReadFile(s.config.Path)
	if err != nil {
		return err
	}

	var stored storeData
	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	s.proxies = make(map[string]v1.ProxyConfigurer)
	s.visitors = make(map[string]v1.VisitorConfigurer)

	for _, tp := range stored.Proxies {
		if tp.ProxyConfigurer != nil {
			proxyCfg := tp.ProxyConfigurer
			name := proxyCfg.GetBaseConfig().Name
			if name == "" {
				return fmt.Errorf("proxy name cannot be empty")
			}
			s.proxies[name] = proxyCfg
		}
	}

	for _, tv := range stored.Visitors {
		if tv.VisitorConfigurer != nil {
			visitorCfg := tv.VisitorConfigurer
			name := visitorCfg.GetBaseConfig().Name
			if name == "" {
				return fmt.Errorf("visitor name cannot be empty")
			}
			s.visitors[name] = visitorCfg
		}
	}

	return nil
}

func (s *StoreSource) saveToFileUnlocked() error {
	stored := storeData{
		Proxies:  make([]v1.TypedProxyConfig, 0, len(s.proxies)),
		Visitors: make([]v1.TypedVisitorConfig, 0, len(s.visitors)),
	}

	for _, p := range s.proxies {
		stored.Proxies = append(stored.Proxies, v1.TypedProxyConfig{ProxyConfigurer: p})
	}
	for _, v := range s.visitors {
		stored.Visitors = append(stored.Visitors, v1.TypedVisitorConfig{VisitorConfigurer: v})
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	dir := filepath.Dir(s.config.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpPath := s.config.Path + ".tmp"

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.config.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func (s *StoreSource) AddProxy(proxy v1.ProxyConfigurer) error {
	if proxy == nil {
		return fmt.Errorf("proxy cannot be nil")
	}

	name := proxy.GetBaseConfig().Name
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.proxies[name]; exists {
		return fmt.Errorf("proxy %q already exists", name)
	}

	s.proxies[name] = proxy

	if err := s.saveToFileUnlocked(); err != nil {
		delete(s.proxies, name)
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) UpdateProxy(proxy v1.ProxyConfigurer) error {
	if proxy == nil {
		return fmt.Errorf("proxy cannot be nil")
	}

	name := proxy.GetBaseConfig().Name
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldProxy, exists := s.proxies[name]
	if !exists {
		return fmt.Errorf("proxy %q does not exist", name)
	}

	s.proxies[name] = proxy

	if err := s.saveToFileUnlocked(); err != nil {
		s.proxies[name] = oldProxy
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) RemoveProxy(name string) error {
	if name == "" {
		return fmt.Errorf("proxy name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldProxy, exists := s.proxies[name]
	if !exists {
		return fmt.Errorf("proxy %q does not exist", name)
	}

	delete(s.proxies, name)

	if err := s.saveToFileUnlocked(); err != nil {
		s.proxies[name] = oldProxy
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) GetProxy(name string) v1.ProxyConfigurer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.proxies[name]
	if !exists {
		return nil
	}
	return p
}

func (s *StoreSource) AddVisitor(visitor v1.VisitorConfigurer) error {
	if visitor == nil {
		return fmt.Errorf("visitor cannot be nil")
	}

	name := visitor.GetBaseConfig().Name
	if name == "" {
		return fmt.Errorf("visitor name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.visitors[name]; exists {
		return fmt.Errorf("visitor %q already exists", name)
	}

	s.visitors[name] = visitor

	if err := s.saveToFileUnlocked(); err != nil {
		delete(s.visitors, name)
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) UpdateVisitor(visitor v1.VisitorConfigurer) error {
	if visitor == nil {
		return fmt.Errorf("visitor cannot be nil")
	}

	name := visitor.GetBaseConfig().Name
	if name == "" {
		return fmt.Errorf("visitor name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldVisitor, exists := s.visitors[name]
	if !exists {
		return fmt.Errorf("visitor %q does not exist", name)
	}

	s.visitors[name] = visitor

	if err := s.saveToFileUnlocked(); err != nil {
		s.visitors[name] = oldVisitor
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) RemoveVisitor(name string) error {
	if name == "" {
		return fmt.Errorf("visitor name cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldVisitor, exists := s.visitors[name]
	if !exists {
		return fmt.Errorf("visitor %q does not exist", name)
	}

	delete(s.visitors, name)

	if err := s.saveToFileUnlocked(); err != nil {
		s.visitors[name] = oldVisitor
		return fmt.Errorf("failed to persist: %w", err)
	}
	return nil
}

func (s *StoreSource) GetVisitor(name string) v1.VisitorConfigurer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, exists := s.visitors[name]
	if !exists {
		return nil
	}
	return v
}

func (s *StoreSource) GetAllProxies() ([]v1.ProxyConfigurer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]v1.ProxyConfigurer, 0, len(s.proxies))
	for _, p := range s.proxies {
		result = append(result, p)
	}
	return result, nil
}

func (s *StoreSource) GetAllVisitors() ([]v1.VisitorConfigurer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]v1.VisitorConfigurer, 0, len(s.visitors))
	for _, v := range s.visitors {
		result = append(result, v)
	}
	return result, nil
}
