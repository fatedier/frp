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

package system

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"
)

func EnableCompatibilityMode() {
	fixTimezone()
	fixDNSResolver()
}

// fixTimezone is used to try our best to fix timezone issue on some Android devices.
func fixTimezone() {
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	loc, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	time.Local = loc
}

// fixDNSResolver will first attempt to resolve google.com to check if the current DNS is available.
// If it is not available, it will default to using 8.8.8.8 as the DNS server.
// This is a workaround for the issue that golang can't get the default DNS servers on Android.
func fixDNSResolver() {
	// First, we attempt to resolve a domain. If resolution is successful, no modifications are necessary.
	// In real-world scenarios, users may have already configured /etc/resolv.conf, or compiled directly
	// in the Android environment instead of using cross-platform compilation, so this issue does not arise.
	if net.DefaultResolver != nil {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := net.DefaultResolver.LookupHost(timeoutCtx, "google.com")
		if err == nil {
			return
		}
	}
	// If the resolution fails, use 8.8.8.8 as the DNS server.
	// Note: If there are other methods to obtain the default DNS servers, the default DNS servers should be used preferentially.
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if addr == "127.0.0.1:53" || addr == "[::1]:53" {
				addr = "8.8.8.8:53"
			}
			var d net.Dialer
			return d.DialContext(ctx, network, addr)
		},
	}
}
