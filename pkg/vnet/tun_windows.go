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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	baseTunName = "utun"
	defaultMTU  = 1420
)

func openTun(_ context.Context, addr string) (tun.Device, error) {
	name, err := findNextTunName(baseTunName)
	if err != nil {
		name = getFallbackTunName(baseTunName, addr)
	}

	// This is for not creating a new entry in Windows device history on each usage
	guid := uuid.NewSHA1(uuid.Nil, []byte(name))
	winGuid, err := windows.GUIDFromString("{" + guid.String() + "}")
	if err != nil {
		return nil, err
	}

	tun.WintunTunnelType = "frpc VirtualNet"
	tunDevice, err := tun.CreateTUNWithRequestedGUID(name, &winGuid, defaultMTU)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN device '%s': %w", name, err)
	}

	actualName, err := tunDevice.Name()
	if err != nil {
		return nil, err
	}

	ip, ipNet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	if err = exec.Command("netsh", "interface", "ipv4", "add", "address", actualName, ip.String(), net.IP(ipNet.Mask).String(), "store=active").Run(); err != nil {
		return nil, err
	}

	routes := []net.IPNet{*ipNet}
	if err = addRoutes(actualName, routes); err != nil {
		return nil, err
	}

	return tunDevice, nil
}

func findNextTunName(basename string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}
	maxSuffix := -1

	for _, iface := range interfaces {
		name := iface.Name
		if strings.HasPrefix(name, basename) {
			suffix := name[len(basename):]
			if suffix == "" {
				continue
			}

			numSuffix, err := strconv.Atoi(suffix)
			if err == nil && numSuffix > maxSuffix {
				maxSuffix = numSuffix
			}
		}
	}

	nextSuffix := maxSuffix + 1
	name := fmt.Sprintf("%s%d", basename, nextSuffix)
	return name, nil
}

// getFallbackTunName generates a deterministic fallback TUN device name
// based on the base name and the provided address string using a hash.
func getFallbackTunName(baseName, addr string) string {
	hasher := sha256.New()
	hasher.Write([]byte(addr))
	hashBytes := hasher.Sum(nil)
	// Use first 4 bytes -> 8 hex chars for brevity, respecting IFNAMSIZ limit.
	shortHash := hex.EncodeToString(hashBytes[:4])
	return fmt.Sprintf("%s%s", baseName, shortHash)
}

func addRoutes(ifName string, routes []net.IPNet) error {
	for _, route := range routes {
		routeStr := route.String()
		if err := exec.Command("netsh", "interface", "ipv4", "add", "route", routeStr, ifName, "store=active").Run(); err != nil {
			return err
		}
	}
	return nil
}
