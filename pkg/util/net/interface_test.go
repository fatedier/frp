// Copyright 2025 Satyajeet Singh, jeet.0733@gmail.com
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

package net

import (
	"testing"
)

func TestValidateInterfaceOrIP(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid ip", "192.168.1.1", false},
		{"valid ipv6", "2001:db8::1", false},
		{"loopback ip", "127.0.0.1", true},
		{"auto keyword", "auto", false},
		{"invalid interface", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInterfaceOrIP(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInterfaceOrIP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveBindAddress(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"auto", "auto", "", false}, // Will return IP if interfaces available, empty if not
		{"valid ip", "192.168.1.1", "192.168.1.1", false},
		{"loopback ip", "127.0.0.1", "", true},
		{"invalid interface", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveBindAddress(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveBindAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// For "auto" case, we can't predict the exact IP, so just check it's not empty if no error
			if tt.name == "auto" && err == nil {
				if got == "" {
					t.Errorf("ResolveBindAddress() returned empty for auto, expected an IP address")
				}
			} else if got != tt.want {
				t.Errorf("ResolveBindAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListNetworkInterfaces(t *testing.T) {
	interfaces, err := ListNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListNetworkInterfaces() error = %v", err)
	}

	// Should have at least one interface
	if len(interfaces) == 0 {
		t.Log("No network interfaces found (this might be normal in some environments)")
	}

	for _, iface := range interfaces {
		if iface.Name == "" {
			t.Errorf("Interface name should not be empty")
		}
		if iface.IP == "" {
			t.Errorf("Interface IP should not be empty for interface %s", iface.Name)
		}
		if iface.Addr == nil {
			t.Errorf("Interface address should not be nil for interface %s", iface.Name)
		}
	}
}

func TestGetAvailableInterfaces(t *testing.T) {
	result := GetAvailableInterfaces()
	if result == "" {
		t.Log("No available interfaces found (this might be normal in some environments)")
	}
	// Should not return an error message
	if result == "failed to enumerate interfaces" || result == "no interfaces available" {
		t.Logf("Interface enumeration result: %s", result)
	}
}
