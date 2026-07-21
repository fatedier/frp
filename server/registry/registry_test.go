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

package registry

import (
	"testing"
	"time"

	clocktesting "k8s.io/utils/clock/testing"

	"github.com/fatedier/frp/pkg/proto/wire"
)

func TestClientRegistryRegisterStoresWireProtocol(t *testing.T) {
	registry := NewClientRegistry()
	key, conflict := registry.Register("user", "client-id", "run-id", "host", "1.0.0", "127.0.0.1", wire.ProtocolV2)
	if conflict {
		t.Fatal("unexpected client conflict")
	}

	info, ok := registry.GetByKey(key)
	if !ok {
		t.Fatalf("client %q not found", key)
	}
	if info.WireProtocol != wire.ProtocolV2 {
		t.Fatalf("wire protocol mismatch, want %q got %q", wire.ProtocolV2, info.WireProtocol)
	}
}

func TestClientRegistryUsesClockForTimestamps(t *testing.T) {
	start := time.Date(2026, time.May, 8, 12, 30, 0, 0, time.UTC)
	clk := clocktesting.NewFakeClock(start)
	registry := newClientRegistryWithClock(clk)

	key, conflict := registry.Register("user", "client-id", "run-id", "host", "1.0.0", "127.0.0.1", wire.ProtocolV2)
	if conflict {
		t.Fatal("unexpected client conflict")
	}

	info, ok := registry.GetByKey(key)
	if !ok {
		t.Fatalf("client %q not found", key)
	}
	if !info.FirstConnectedAt.Equal(start) {
		t.Fatalf("first connected time mismatch, want %s got %s", start, info.FirstConnectedAt)
	}
	if !info.LastConnectedAt.Equal(start) {
		t.Fatalf("last connected time mismatch, want %s got %s", start, info.LastConnectedAt)
	}

	disconnectedAt := start.Add(time.Minute)
	clk.SetTime(disconnectedAt)
	registry.MarkOfflineByRunID("run-id")

	info, ok = registry.GetByKey(key)
	if !ok {
		t.Fatalf("client %q not found after disconnect", key)
	}
	if !info.DisconnectedAt.Equal(disconnectedAt) {
		t.Fatalf("disconnected time mismatch, want %s got %s", disconnectedAt, info.DisconnectedAt)
	}
}

func TestClientRegistryControlIDPreventsStaleOffline(t *testing.T) {
	registry := NewClientRegistry()
	key, conflict := registry.RegisterWithControlID(
		"user", "client-id", "run-id", "old-host", "1.0.0", "127.0.0.1", wire.ProtocolV1, 1,
	)
	if conflict {
		t.Fatal("unexpected client conflict")
	}
	_, conflict = registry.RegisterWithControlID(
		"user", "client-id", "run-id", "new-host", "1.0.1", "127.0.0.2", wire.ProtocolV2, 2,
	)
	if conflict {
		t.Fatal("same run ID replacement should not conflict")
	}

	registry.MarkOfflineByRunIDAndControlID("run-id", 1)
	info, ok := registry.GetByKey(key)
	if !ok {
		t.Fatalf("client %q not found", key)
	}
	if !info.Online || info.ControlID != 2 || info.Hostname != "new-host" {
		t.Fatalf("stale offline changed current generation: %+v", info)
	}

	registry.MarkOfflineByRunIDAndControlID("run-id", 2)
	info, ok = registry.GetByKey(key)
	if !ok {
		t.Fatalf("client %q not found after disconnect", key)
	}
	if info.Online || info.ControlID != 0 || info.RunID != "" {
		t.Fatalf("current generation was not marked offline: %+v", info)
	}
}

func TestClientRegistryClientIDConflictSemantics(t *testing.T) {
	registry := NewClientRegistry()
	_, conflict := registry.RegisterWithControlID(
		"user", "client-id", "run-one", "host", "1.0.0", "127.0.0.1", wire.ProtocolV1, 1,
	)
	if conflict {
		t.Fatal("unexpected initial client conflict")
	}
	_, conflict = registry.RegisterWithControlID(
		"user", "client-id", "run-two", "host", "1.0.0", "127.0.0.2", wire.ProtocolV1, 2,
	)
	if !conflict {
		t.Fatal("different online run IDs with the same explicit client ID must conflict")
	}

	registry.MarkOfflineByRunIDAndControlID("run-one", 1)
	_, conflict = registry.RegisterWithControlID(
		"user", "client-id", "run-two", "host", "1.0.0", "127.0.0.2", wire.ProtocolV1, 2,
	)
	if conflict {
		t.Fatal("offline explicit client ID should be reusable")
	}
}

func TestClientRegistrySameRunIDMovesBetweenClientKeys(t *testing.T) {
	registry := NewClientRegistry()
	oldKey, conflict := registry.RegisterWithControlID(
		"user", "old-client", "run-id", "old-host", "1.0.0", "127.0.0.1", wire.ProtocolV1, 1,
	)
	if conflict {
		t.Fatal("unexpected initial client conflict")
	}
	newKey, conflict := registry.RegisterWithControlID(
		"user", "new-client", "run-id", "new-host", "1.0.1", "127.0.0.2", wire.ProtocolV2, 2,
	)
	if conflict {
		t.Fatal("same run ID moving to a new client key should not conflict")
	}

	oldInfo, ok := registry.GetByKey(oldKey)
	if !ok {
		t.Fatalf("old explicit client %q should remain as offline history", oldKey)
	}
	if oldInfo.Online || oldInfo.RunID != "" || oldInfo.ControlID != 0 {
		t.Fatalf("old client key remained online: %+v", oldInfo)
	}
	newInfo, ok := registry.GetByKey(newKey)
	if !ok || !newInfo.Online || newInfo.RunID != "run-id" || newInfo.ControlID != 2 {
		t.Fatalf("new client key was not registered: %+v", newInfo)
	}
}
