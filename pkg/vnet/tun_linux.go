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

	"github.com/vishvananda/netlink"
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

	ifn, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	ip, cidr, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}
	if err := netlink.AddrAdd(link, &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: cidr.Mask,
		},
	}); err != nil {
		return nil, err
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, err
	}

	if err = addRoutes(ifn, cidr); err != nil {
		return nil, err
	}
	return dev, nil
}

func addRoutes(ifn *net.Interface, cidr *net.IPNet) error {
	r := netlink.Route{
		Dst:       cidr,
		LinkIndex: ifn.Index,
	}
	if err := netlink.RouteReplace(&r); err != nil {
		return fmt.Errorf("add route to %v error: %v", r.Dst, err)
	}
	return nil
}
