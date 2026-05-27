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

package nathole

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputePortDelta(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []string
		expected int
	}{
		{
			name:     "consistent delta of 2",
			addrs:    []string{"1.2.3.4:40000", "1.2.3.4:40002", "1.2.3.4:40004"},
			expected: 2,
		},
		{
			name:     "consistent delta of 1",
			addrs:    []string{"1.2.3.4:5000", "1.2.3.4:5001", "1.2.3.4:5002", "1.2.3.4:5003"},
			expected: 1,
		},
		{
			name:     "unordered but consistent delta",
			addrs:    []string{"1.2.3.4:40004", "1.2.3.4:40000", "1.2.3.4:40002"},
			expected: 2,
		},
		{
			name:     "inconsistent deltas",
			addrs:    []string{"1.2.3.4:40000", "1.2.3.4:40002", "1.2.3.4:40007"},
			expected: 0,
		},
		{
			name:     "single address",
			addrs:    []string{"1.2.3.4:40000"},
			expected: 0,
		},
		{
			name:     "empty addresses",
			addrs:    []string{},
			expected: 0,
		},
		{
			name:     "same port (no change)",
			addrs:    []string{"1.2.3.4:40000", "1.2.3.4:40000"},
			expected: 0,
		},
		{
			name:     "delta too large (over 100)",
			addrs:    []string{"1.2.3.4:40000", "1.2.3.4:40200", "1.2.3.4:40400"},
			expected: 0,
		},
		{
			name:     "delta exactly 100",
			addrs:    []string{"1.2.3.4:1000", "1.2.3.4:1100", "1.2.3.4:1200"},
			expected: 100,
		},
		{
			name:     "invalid address format",
			addrs:    []string{"not-an-address", "also-bad"},
			expected: 0,
		},
		{
			name:     "mixed valid and invalid",
			addrs:    []string{"1.2.3.4:5000", "bad-addr", "1.2.3.4:5002"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePortDelta(tt.addrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPredictNextPorts(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []string
		delta    int
		count    int
		expected []string
	}{
		{
			name:  "predict 3 ports with delta 2",
			addrs: []string{"1.2.3.4:40000"},
			delta: 2,
			count: 3,
			expected: []string{
				"1.2.3.4:40002",
				"1.2.3.4:40004",
				"1.2.3.4:40006",
			},
		},
		{
			name:  "predict from multiple addresses",
			addrs: []string{"1.2.3.4:5000", "5.6.7.8:6000"},
			delta: 1,
			count: 2,
			expected: []string{
				"1.2.3.4:5001",
				"1.2.3.4:5002",
				"5.6.7.8:6001",
				"5.6.7.8:6002",
			},
		},
		{
			name:     "port overflow (near 65535)",
			addrs:    []string{"1.2.3.4:65530"},
			delta:    10,
			count:    3,
			expected: []string{},
		},
		{
			name:     "empty addresses",
			addrs:    []string{},
			delta:    2,
			count:    3,
			expected: []string{},
		},
		{
			name:     "invalid address",
			addrs:    []string{"not-valid"},
			delta:    2,
			count:    3,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predictNextPorts(tt.addrs, tt.delta, tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []string
		expected []int
	}{
		{
			name:     "valid addresses",
			addrs:    []string{"1.2.3.4:5000", "5.6.7.8:6000"},
			expected: []int{5000, 6000},
		},
		{
			name:     "invalid addresses skipped",
			addrs:    []string{"bad", "1.2.3.4:7000", "also-bad"},
			expected: []int{7000},
		},
		{
			name:     "empty",
			addrs:    []string{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePorts(tt.addrs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
