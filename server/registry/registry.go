// Copyright 2025 The frp Authors
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

package registry

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/utils/clock"
)

// ClientInfo captures metadata about a connected frpc instance.
type ClientInfo struct {
	Key              string
	User             string
	RawClientID      string
	RunID            string
	ControlID        uint64
	Hostname         string
	IP               string
	Version          string
	WireProtocol     string
	FirstConnectedAt time.Time
	LastConnectedAt  time.Time
	DisconnectedAt   time.Time
	Online           bool
}

// ClientRegistry keeps track of active clients keyed by "{user}.{clientID}" (runID fallback when raw clientID is empty).
// Entries without an explicit raw clientID are removed on disconnect to avoid stale offline records.
type ClientRegistry struct {
	mu       sync.RWMutex
	clients  map[string]*ClientInfo
	runIndex map[string]string
	clock    clock.PassiveClock
}

func NewClientRegistry() *ClientRegistry {
	return newClientRegistryWithClock(clock.RealClock{})
}

func newClientRegistryWithClock(clk clock.PassiveClock) *ClientRegistry {
	if clk == nil {
		clk = clock.RealClock{}
	}
	return &ClientRegistry{
		clients:  make(map[string]*ClientInfo),
		runIndex: make(map[string]string),
		clock:    clk,
	}
}

// Register stores/updates metadata for a client and returns the registry key plus whether it conflicts with an online client.
func (cr *ClientRegistry) Register(user, rawClientID, runID, hostname, version, remoteAddr, wireProtocol string) (key string, conflict bool) {
	return cr.RegisterWithControlID(user, rawClientID, runID, hostname, version, remoteAddr, wireProtocol, 0)
}

// RegisterWithControlID is the generation-aware form used by ControlManager.
// A control ID is process-local and prevents an older control generation from
// changing the registry entry now owned by a newer generation with the same run ID.
func (cr *ClientRegistry) RegisterWithControlID(
	user, rawClientID, runID, hostname, version, remoteAddr, wireProtocol string,
	controlID uint64,
) (key string, conflict bool) {
	if runID == "" {
		return "", false
	}

	effectiveID := rawClientID
	if effectiveID == "" {
		effectiveID = runID
	}
	key = cr.composeClientKey(user, effectiveID)
	enforceUnique := rawClientID != ""

	now := cr.clock.Now()
	cr.mu.Lock()
	defer cr.mu.Unlock()

	info, exists := cr.clients[key]
	if enforceUnique && exists && info.Online && info.RunID != "" && info.RunID != runID {
		return key, true
	}
	if previousKey, ok := cr.runIndex[runID]; ok && previousKey != key {
		if previous, ok := cr.clients[previousKey]; ok && previous.RunID == runID {
			if previous.RawClientID == "" {
				delete(cr.clients, previousKey)
			} else {
				setClientOffline(previous, now)
			}
		}
		delete(cr.runIndex, runID)
	}

	if !exists {
		info = &ClientInfo{
			Key:              key,
			User:             user,
			FirstConnectedAt: now,
		}
		cr.clients[key] = info
	} else if info.RunID != "" {
		delete(cr.runIndex, info.RunID)
	}

	info.RawClientID = rawClientID
	info.RunID = runID
	info.ControlID = controlID
	info.Hostname = hostname
	info.IP = remoteAddr
	info.Version = version
	info.WireProtocol = wireProtocol
	if info.FirstConnectedAt.IsZero() {
		info.FirstConnectedAt = now
	}
	info.LastConnectedAt = now
	info.DisconnectedAt = time.Time{}
	info.Online = true

	cr.runIndex[runID] = key
	return key, false
}

// MarkOfflineByRunID marks the client as offline when the corresponding control disconnects.
func (cr *ClientRegistry) MarkOfflineByRunID(runID string) {
	cr.markOfflineByRunID(runID, 0, false)
}

// MarkOfflineByRunIDAndControlID marks a client offline only when the registry
// entry still belongs to the supplied control generation.
func (cr *ClientRegistry) MarkOfflineByRunIDAndControlID(runID string, controlID uint64) {
	cr.markOfflineByRunID(runID, controlID, true)
}

func (cr *ClientRegistry) markOfflineByRunID(runID string, controlID uint64, matchControlID bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	key, ok := cr.runIndex[runID]
	if !ok {
		return
	}
	if info, ok := cr.clients[key]; ok && info.RunID == runID && (!matchControlID || info.ControlID == controlID) {
		if info.RawClientID == "" {
			delete(cr.clients, key)
		} else {
			setClientOffline(info, cr.clock.Now())
		}
	}
	if info, ok := cr.clients[key]; !ok || info.RunID != runID {
		delete(cr.runIndex, runID)
	}
}

func setClientOffline(info *ClientInfo, now time.Time) {
	info.RunID = ""
	info.ControlID = 0
	info.Online = false
	info.DisconnectedAt = now
}

// List returns a snapshot of all known clients.
func (cr *ClientRegistry) List() []ClientInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	result := make([]ClientInfo, 0, len(cr.clients))
	for _, info := range cr.clients {
		result = append(result, *info)
	}
	return result
}

// GetByKey retrieves a client by its composite key ({user}.{clientID} with runID fallback).
func (cr *ClientRegistry) GetByKey(key string) (ClientInfo, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	info, ok := cr.clients[key]
	if !ok {
		return ClientInfo{}, false
	}
	return *info, true
}

// ClientID returns the resolved client identifier for external use.
func (info ClientInfo) ClientID() string {
	if info.RawClientID != "" {
		return info.RawClientID
	}
	return info.RunID
}

func (cr *ClientRegistry) composeClientKey(user, id string) string {
	switch {
	case user == "":
		return id
	case id == "":
		return user
	default:
		return fmt.Sprintf("%s.%s", user, id)
	}
}
