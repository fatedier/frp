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

package vnet

import (
	"context"
	"fmt"
	"net"
	"os/exec"

	"golang.zx2c4.com/wireguard/tun"
)

const (
	defaultTunName = "utun"
	defaultMTU     = 1420
)

func openTun(_ context.Context, addr string) (tun.Device, error) {
	dev, err := tun.CreateTUN(defaultTunName, defaultMTU)
	if err != nil {
		return nil, err
	}

	name, err := dev.Name()
	if err != nil {
		return nil, err
	}

	ip, ipNet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	// Calculate a peer IP for the point-to-point tunnel
	peerIP := generatePeerIP(ip)

	// Configure the interface with proper point-to-point addressing
	if err = exec.Command("ifconfig", name, "inet", ip.String(), peerIP.String(), "mtu", fmt.Sprint(defaultMTU), "up").Run(); err != nil {
		return nil, err
	}

	// Add default route for the tunnel subnet
	routes := []net.IPNet{*ipNet}
	if err = addRoutes(name, routes); err != nil {
		return nil, err
	}
	return dev, nil
}

// generatePeerIP creates a peer IP for the point-to-point tunnel
// by incrementing the last octet of the IP
func generatePeerIP(ip net.IP) net.IP {
	// Make a copy to avoid modifying the original
	peerIP := make(net.IP, len(ip))
	copy(peerIP, ip)

	// Increment the last octet
	peerIP[len(peerIP)-1]++

	return peerIP
}

// addRoutes configures system routes for the TUN interface
func addRoutes(ifName string, routes []net.IPNet) error {
	for _, route := range routes {
		routeStr := route.String()
		if err := exec.Command("route", "add", "-net", routeStr, "-interface", ifName).Run(); err != nil {
			return err
		}
	}
	return nil
}
