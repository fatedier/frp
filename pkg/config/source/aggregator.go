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
	"errors"
	"fmt"
	"sort"
	"sync"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type sourceEntry struct {
	source Source
}

type Aggregator struct {
	mu sync.RWMutex

	configSource *ConfigSource
	storeSource  *StoreSource
}

func NewAggregator(configSource *ConfigSource) *Aggregator {
	if configSource == nil {
		configSource = NewConfigSource()
	}
	return &Aggregator{
		configSource: configSource,
	}
}

func (a *Aggregator) SetStoreSource(storeSource *StoreSource) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.storeSource = storeSource
}

func (a *Aggregator) ConfigSource() *ConfigSource {
	return a.configSource
}

func (a *Aggregator) StoreSource() *StoreSource {
	return a.storeSource
}

func (a *Aggregator) getSourcesLocked() []sourceEntry {
	sources := make([]sourceEntry, 0, 2)
	if a.configSource != nil {
		sources = append(sources, sourceEntry{
			source: a.configSource,
		})
	}
	if a.storeSource != nil {
		sources = append(sources, sourceEntry{
			source: a.storeSource,
		})
	}
	return sources
}

func (a *Aggregator) Load() ([]v1.ProxyConfigurer, []v1.VisitorConfigurer, error) {
	a.mu.RLock()
	entries := a.getSourcesLocked()
	a.mu.RUnlock()

	if len(entries) == 0 {
		return nil, nil, errors.New("no sources configured")
	}

	proxyMap := make(map[string]v1.ProxyConfigurer)
	visitorMap := make(map[string]v1.VisitorConfigurer)

	for _, entry := range entries {
		proxies, visitors, err := entry.source.Load()
		if err != nil {
			return nil, nil, fmt.Errorf("load source: %w", err)
		}
		for _, p := range proxies {
			proxyMap[p.GetBaseConfig().Name] = p
		}
		for _, v := range visitors {
			visitorMap[v.GetBaseConfig().Name] = v
		}
	}
	proxies, visitors := a.mapsToSortedSlices(proxyMap, visitorMap)
	return proxies, visitors, nil
}

func (a *Aggregator) mapsToSortedSlices(
	proxyMap map[string]v1.ProxyConfigurer,
	visitorMap map[string]v1.VisitorConfigurer,
) ([]v1.ProxyConfigurer, []v1.VisitorConfigurer) {
	proxies := make([]v1.ProxyConfigurer, 0, len(proxyMap))
	for _, p := range proxyMap {
		proxies = append(proxies, p)
	}
	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].GetBaseConfig().Name < proxies[j].GetBaseConfig().Name
	})

	visitors := make([]v1.VisitorConfigurer, 0, len(visitorMap))
	for _, v := range visitorMap {
		visitors = append(visitors, v)
	}
	sort.Slice(visitors, func(i, j int) bool {
		return visitors[i].GetBaseConfig().Name < visitors[j].GetBaseConfig().Name
	})

	return proxies, visitors
}
