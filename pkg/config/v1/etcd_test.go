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

package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEtcdConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   EtcdConfig
		expected bool
	}{
		{
			name:     "empty config",
			config:   EtcdConfig{},
			expected: false,
		},
		{
			name: "only endpoints",
			config: EtcdConfig{
				Endpoints: []string{"127.0.0.1:2379"},
			},
			expected: false,
		},
		{
			name: "only region",
			config: EtcdConfig{
				Region: "us-east",
			},
			expected: false,
		},
		{
			name: "both endpoints and region",
			config: EtcdConfig{
				Endpoints: []string{"127.0.0.1:2379"},
				Region:    "us-east",
			},
			expected: true,
		},
		{
			name: "multiple endpoints with region",
			config: EtcdConfig{
				Endpoints: []string{"127.0.0.1:2379", "127.0.0.1:2380"},
				Region:    "us-east",
			},
			expected: true,
		},
		{
			name: "empty endpoints array with region",
			config: EtcdConfig{
				Endpoints: []string{},
				Region:    "us-east",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenConfig_BasicFields(t *testing.T) {
	config := TokenConfig{
		Token:                   "test-token-123",
		Region:                  "us-east",
		Enabled:                 true,
		MaxPortsPerClient:       5,
		Description:             "Test token",
		TrafficReportIntervalMB: 50,
	}

	assert.Equal(t, "test-token-123", config.Token)
	assert.Equal(t, "us-east", config.Region)
	assert.True(t, config.Enabled)
	assert.Equal(t, int64(5), config.MaxPortsPerClient)
	assert.Equal(t, "Test token", config.Description)
	assert.Equal(t, int64(50), config.TrafficReportIntervalMB)
}

func TestTokenConfig_ParseFromJSON(t *testing.T) {
	jsonStr := `{
		"token": "my-secret-token",
		"region": "chengdu",
		"allowPorts": [{"start": 8000, "end": 9000}],
		"bandwidthLimit": "10MB",
		"maxPortsPerClient": 5,
		"enabled": true,
		"description": "User token",
		"trafficReportIntervalMB": 100
	}`

	var config TokenConfig
	err := json.Unmarshal([]byte(jsonStr), &config)
	require.NoError(t, err)

	assert.Equal(t, "my-secret-token", config.Token)
	assert.Equal(t, "chengdu", config.Region)
	assert.True(t, config.Enabled)
	assert.Equal(t, int64(5), config.MaxPortsPerClient)
	assert.Equal(t, "User token", config.Description)
	assert.Equal(t, int64(100), config.TrafficReportIntervalMB)
	assert.Len(t, config.AllowPorts, 1)
	assert.Equal(t, 8000, config.AllowPorts[0].Start)
	assert.Equal(t, 9000, config.AllowPorts[0].End)
}
