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
