package server

import (
	"fmt"
	"sync"
	"time"
)

// ClientInfo captures metadata about a connected frpc instance.
type ClientInfo struct {
	Key              string
	User             string
	ClientID         string
	RunID            string
	Hostname         string
	IP               string
	FirstConnectedAt time.Time
	LastConnectedAt  time.Time
	DisconnectedAt   time.Time
	Online           bool
}

// ClientRegistry keeps track of active clients keyed by "{user}.{clientID}" (or runID if clientID is empty).
// Entries without an explicit clientID are removed on disconnect to avoid stale offline records.
type ClientRegistry struct {
	mu       sync.RWMutex
	clients  map[string]*ClientInfo
	runIndex map[string]string
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients:  make(map[string]*ClientInfo),
		runIndex: make(map[string]string),
	}
}

// Register stores/updates metadata for a client and returns the registry key plus whether it conflicts with an online client.
func (cr *ClientRegistry) Register(user, clientID, runID, hostname, remoteAddr string) (key string, conflict bool) {
	if runID == "" {
		return "", false
	}

	effectiveID := clientID
	if effectiveID == "" {
		effectiveID = runID
	}
	key = cr.composeClientKey(user, effectiveID)
	enforceUnique := clientID != ""

	now := time.Now()
	cr.mu.Lock()
	defer cr.mu.Unlock()

	info, exists := cr.clients[key]
	if enforceUnique && exists && info.Online && info.RunID != "" && info.RunID != runID {
		return key, true
	}

	if !exists {
		info = &ClientInfo{
			Key:              key,
			User:             user,
			ClientID:         clientID,
			FirstConnectedAt: now,
		}
		cr.clients[key] = info
	} else if info.RunID != "" {
		delete(cr.runIndex, info.RunID)
	}

	info.RunID = runID
	info.Hostname = hostname
	info.IP = remoteAddr
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
	cr.mu.Lock()
	defer cr.mu.Unlock()

	key, ok := cr.runIndex[runID]
	if !ok {
		return
	}
	if info, ok := cr.clients[key]; ok && info.RunID == runID {
		if info.ClientID == "" {
			delete(cr.clients, key)
		} else {
			info.RunID = ""
			info.Online = false
			now := time.Now()
			info.DisconnectedAt = now
		}
	}
	delete(cr.runIndex, runID)
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

// GetByKey retrieves a client by its composite key ({user}.{clientID} or runID fallback).
func (cr *ClientRegistry) GetByKey(key string) (ClientInfo, bool) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	info, ok := cr.clients[key]
	if !ok {
		return ClientInfo{}, false
	}
	return *info, true
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
