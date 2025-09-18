package net

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIPValidator(t *testing.T) {
	tests := []struct {
		name        string
		allowedIPs  []string
		expectError bool
		expectNil   bool
	}{
		{
			name:        "empty list",
			allowedIPs:  []string{},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "nil list",
			allowedIPs:  nil,
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "valid single IPv4",
			allowedIPs:  []string{"127.0.0.1"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "valid IPv4 CIDR",
			allowedIPs:  []string{"192.168.1.0/24"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "valid IPv6",
			allowedIPs:  []string{"::1"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "valid IPv6 CIDR",
			allowedIPs:  []string{"2001:db8::/32"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "mixed valid IPs",
			allowedIPs:  []string{"127.0.0.1", "192.168.1.0/24", "::1"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "invalid IP",
			allowedIPs:  []string{"invalid.ip"},
			expectError: true,
			expectNil:   true,
		},
		{
			name:        "invalid CIDR",
			allowedIPs:  []string{"192.168.1.0/33"},
			expectError: true,
			expectNil:   true,
		},
		{
			name:        "empty string in list",
			allowedIPs:  []string{"127.0.0.1", "", "192.168.1.1"},
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "whitespace in list",
			allowedIPs:  []string{"  127.0.0.1  ", "192.168.1.1"},
			expectError: false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewIPValidator(tt.allowedIPs)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestIPValidator_IsAllowed(t *testing.T) {
	tests := []struct {
		name       string
		allowedIPs []string
		testIP     string
		expected   bool
	}{
		{
			name:       "empty whitelist allows all",
			allowedIPs: []string{},
			testIP:     "127.0.0.1",
			expected:   true,
		},
		{
			name:       "nil whitelist allows all",
			allowedIPs: nil,
			testIP:     "192.168.1.1",
			expected:   true,
		},
		{
			name:       "exact IPv4 match",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "127.0.0.1",
			expected:   true,
		},
		{
			name:       "IPv4 no match",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "192.168.1.1",
			expected:   false,
		},
		{
			name:       "IPv4 CIDR match",
			allowedIPs: []string{"192.168.1.0/24"},
			testIP:     "192.168.1.100",
			expected:   true,
		},
		{
			name:       "IPv4 CIDR no match",
			allowedIPs: []string{"192.168.1.0/24"},
			testIP:     "192.168.2.1",
			expected:   false,
		},
		{
			name:       "IPv6 exact match",
			allowedIPs: []string{"::1"},
			testIP:     "::1",
			expected:   true,
		},
		{
			name:       "IPv6 no match",
			allowedIPs: []string{"::1"},
			testIP:     "2001:db8::1",
			expected:   false,
		},
		{
			name:       "IPv6 CIDR match",
			allowedIPs: []string{"2001:db8::/32"},
			testIP:     "2001:db8::1",
			expected:   true,
		},
		{
			name:       "multiple IPs first match",
			allowedIPs: []string{"127.0.0.1", "192.168.1.0/24"},
			testIP:     "127.0.0.1",
			expected:   true,
		},
		{
			name:       "multiple IPs second match",
			allowedIPs: []string{"127.0.0.1", "192.168.1.0/24"},
			testIP:     "192.168.1.50",
			expected:   true,
		},
		{
			name:       "multiple IPs no match",
			allowedIPs: []string{"127.0.0.1", "192.168.1.0/24"},
			testIP:     "10.0.0.1",
			expected:   false,
		},
		{
			name:       "IP with port",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "127.0.0.1:8080",
			expected:   true,
		},
		{
			name:       "IP with port no match",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "192.168.1.1:8080",
			expected:   false,
		},
		{
			name:       "invalid IP format",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "invalid.ip",
			expected:   false,
		},
		{
			name:       "empty test IP",
			allowedIPs: []string{"127.0.0.1"},
			testIP:     "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewIPValidator(tt.allowedIPs)
			require.NoError(t, err)
			require.NotNil(t, validator)

			result := validator.IsAllowed(tt.testIP)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIPValidator_GetAllowedIPs(t *testing.T) {
	tests := []struct {
		name       string
		allowedIPs []string
		expected   []string
	}{
		{
			name:       "empty list",
			allowedIPs: []string{},
			expected:   []string{},
		},
		{
			name:       "nil list",
			allowedIPs: nil,
			expected:   []string{},
		},
		{
			name:       "single IP",
			allowedIPs: []string{"127.0.0.1"},
			expected:   []string{"127.0.0.1/32"},
		},
		{
			name:       "CIDR block",
			allowedIPs: []string{"192.168.1.0/24"},
			expected:   []string{"192.168.1.0/24"},
		},
		{
			name:       "mixed IPs",
			allowedIPs: []string{"127.0.0.1", "192.168.1.0/24"},
			expected:   []string{"127.0.0.1/32", "192.168.1.0/24"},
		},
		{
			name:       "IPv6",
			allowedIPs: []string{"::1"},
			expected:   []string{"::1/128"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewIPValidator(tt.allowedIPs)
			require.NoError(t, err)
			require.NotNil(t, validator)

			result := validator.GetAllowedIPs()
			assert.Equal(t, tt.expected, result)
		})
	}
}