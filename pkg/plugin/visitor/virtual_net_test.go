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

//go:build !frps

package visitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestVirtualNetReconnectDelay verifies the documented exponential backoff and
// ensures large error counts remain capped instead of overflowing to zero.
func TestVirtualNetReconnectDelay(t *testing.T) {
	tests := []struct {
		name              string
		consecutiveErrors int
		want              time.Duration
	}{
		{name: "first error", consecutiveErrors: 1, want: 60 * time.Second},
		{name: "second error", consecutiveErrors: 2, want: 120 * time.Second},
		{name: "third error", consecutiveErrors: 3, want: 240 * time.Second},
		{name: "fourth error", consecutiveErrors: 4, want: 300 * time.Second},
		{name: "shift width boundary", consecutiveErrors: 64, want: 300 * time.Second},
		{name: "observed retry storm", consecutiveErrors: 329769, want: 300 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, virtualNetReconnectDelay(tt.consecutiveErrors))
		})
	}
}
