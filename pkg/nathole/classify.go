// Copyright 2023 The frp Authors
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
	"fmt"
	"net"
)

const (
	EasyNAT = "EasyNAT"
	HardNAT = "HardNAT"

	BehaviorNoChange    = "BehaviorNoChange"
	BehaviorIPChanged   = "BehaviorIPChanged"
	BehaviorPortChanged = "BehaviorPortChanged"
	BehaviorBothChanged = "BehaviorBothChanged"
)

// ClassifyNATType classify NAT type by given addresses.
func ClassifyNATType(addresses []string) (string, string, error) {
	if len(addresses) <= 1 {
		return "", "", fmt.Errorf("not enough addresses")
	}
	ipChanged := false
	portChanged := false

	var baseIP, basePort string
	for _, addr := range addresses {
		ip, port, err := net.SplitHostPort(addr)
		if err != nil {
			return "", "", err
		}
		if baseIP == "" {
			baseIP = ip
			basePort = port
			continue
		}

		if baseIP != ip {
			ipChanged = true
		}
		if basePort != port {
			portChanged = true
		}

		if ipChanged && portChanged {
			break
		}
	}

	switch {
	case ipChanged && portChanged:
		return HardNAT, BehaviorBothChanged, nil
	case ipChanged:
		return HardNAT, BehaviorIPChanged, nil
	case portChanged:
		return HardNAT, BehaviorPortChanged, nil
	default:
		return EasyNAT, BehaviorNoChange, nil
	}
}
