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
	"fmt"
	"net"
	"strings"
)

// InterfaceInfo represents information about a network interface
type InterfaceInfo struct {
	Name string
	IP   string
	Addr net.Addr
}

// GetInterfaceIP resolves an interface name to its primary IPv4 address
func GetInterfaceIP(interfaceName string) (string, error) {
	if interfaceName == "" {
		return "", fmt.Errorf("interface name cannot be empty")
	}

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("interface '%s' not found: %v", interfaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for interface '%s': %v", interfaceName, err)
	}

	// Look for IPv4 address first
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}

	// If no IPv4, look for IPv6
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To16(); ip != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("interface '%s' has no valid IP address assigned", interfaceName)
}

// ListNetworkInterfaces returns all available network interfaces with their IP addresses
func ListNetworkInterfaces() ([]InterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network interfaces: %v", err)
	}

	var result []InterfaceInfo
	for _, iface := range interfaces {
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue // Skip interfaces with address errors
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				// Skip loopback addresses
				if ipnet.IP.IsLoopback() {
					continue
				}

				result = append(result, InterfaceInfo{
					Name: iface.Name,
					IP:   ipnet.IP.String(),
					Addr: addr,
				})
			}
		}
	}

	return result, nil
}

// GetFirstNonLoopbackIP returns the first available non-loopback IPv4 address
func GetFirstNonLoopbackIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to enumerate network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() {
					return ip.String(), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// ValidateInterfaceOrIP validates if the given value is a valid interface name or IP address
func ValidateInterfaceOrIP(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(value); ip != nil {
		if ip.IsLoopback() {
			return fmt.Errorf("loopback IP address '%s' is not allowed", value)
		}
		return nil
	}

	// Check if it's "auto" (special keyword)
	if value == "auto" {
		return nil
	}

	// Check if it's a valid interface name
	if _, err := net.InterfaceByName(value); err != nil {
		return fmt.Errorf("'%s' is not a valid interface name or IP address", value)
	}

	return nil
}

// ResolveBindAddress resolves the binding address from interface name or IP address
func ResolveBindAddress(value string) (string, error) {
	if value == "" {
		return "", nil // No binding specified
	}

	if value == "auto" {
		return GetFirstNonLoopbackIP()
	}

	// Check if it's already an IP address
	if ip := net.ParseIP(value); ip != nil {
		if ip.IsLoopback() {
			return "", fmt.Errorf("loopback IP address '%s' is not allowed", value)
		}
		return value, nil
	}

	// Treat as interface name
	return GetInterfaceIP(value)
}

// GetAvailableInterfaces returns a formatted string of available interfaces for error messages
func GetAvailableInterfaces() string {
	interfaces, err := ListNetworkInterfaces()
	if err != nil {
		return "failed to enumerate interfaces"
	}

	if len(interfaces) == 0 {
		return "no interfaces available"
	}

	var names []string
	seen := make(map[string]bool)
	for _, iface := range interfaces {
		if !seen[iface.Name] {
			names = append(names, iface.Name)
			seen[iface.Name] = true
		}
	}

	return strings.Join(names, ", ")
}

