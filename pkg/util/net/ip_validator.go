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

package net

import (
	"fmt"
	"net"
	"strings"
)

type IPValidator struct {
	allowedIPs []*net.IPNet
}

func NewIPValidator(allowedClientIPs []string) (*IPValidator, error) {
	if len(allowedClientIPs) == 0 {
		return &IPValidator{allowedIPs: nil}, nil
	}

	validator := &IPValidator{
		allowedIPs: make([]*net.IPNet, 0, len(allowedClientIPs)),
	}

	for _, ipStr := range allowedClientIPs {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		if strings.Contains(ipStr, "/") {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR block %s: %v", ipStr, err)
			}
			validator.allowedIPs = append(validator.allowedIPs, ipNet)
		} else {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address: %s", ipStr)
			}
			
			var ipNet *net.IPNet
			if ip.To4() != nil {
				ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
			} else {
				ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
			}
			validator.allowedIPs = append(validator.allowedIPs, ipNet)
		}
	}

	return validator, nil
}

func (v *IPValidator) IsAllowed(ipStr string) bool {
	if len(v.allowedIPs) == 0 {
		return true
	}

	host, _, err := net.SplitHostPort(ipStr)
	if err != nil {
		host = ipStr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, allowedNet := range v.allowedIPs {
		if allowedNet.Contains(ip) {
			return true
		}
	}

	return false
}

func (v *IPValidator) GetAllowedIPs() []string {
	if len(v.allowedIPs) == 0 {
		return []string{}
	}

	result := make([]string, len(v.allowedIPs))
	for i, ipNet := range v.allowedIPs {
		result[i] = ipNet.String()
	}
	return result
}