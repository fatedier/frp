// Copyright 2014 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"net/url"
	"reflect"
	"testing"
)

var parseURLTests = []struct {
	s   string
	u   *url.URL
	rui string
}{
	{"ws://example.com/", &url.URL{Scheme: "ws", Host: "example.com", Opaque: "/"}, "/"},
	{"ws://example.com", &url.URL{Scheme: "ws", Host: "example.com", Opaque: "/"}, "/"},
	{"ws://example.com:7777/", &url.URL{Scheme: "ws", Host: "example.com:7777", Opaque: "/"}, "/"},
	{"wss://example.com/", &url.URL{Scheme: "wss", Host: "example.com", Opaque: "/"}, "/"},
	{"wss://example.com/a/b", &url.URL{Scheme: "wss", Host: "example.com", Opaque: "/a/b"}, "/a/b"},
	{"ss://example.com/a/b", nil, ""},
	{"ws://webmaster@example.com/", nil, ""},
	{"wss://example.com/a/b?x=y", &url.URL{Scheme: "wss", Host: "example.com", Opaque: "/a/b", RawQuery: "x=y"}, "/a/b?x=y"},
	{"wss://example.com?x=y", &url.URL{Scheme: "wss", Host: "example.com", Opaque: "/", RawQuery: "x=y"}, "/?x=y"},
}

func TestParseURL(t *testing.T) {
	for _, tt := range parseURLTests {
		u, err := parseURL(tt.s)
		if tt.u != nil && err != nil {
			t.Errorf("parseURL(%q) returned error %v", tt.s, err)
			continue
		}
		if tt.u == nil {
			if err == nil {
				t.Errorf("parseURL(%q) did not return error", tt.s)
			}
			continue
		}
		if !reflect.DeepEqual(u, tt.u) {
			t.Errorf("parseURL(%q) = %v, want %v", tt.s, u, tt.u)
			continue
		}
		if u.RequestURI() != tt.rui {
			t.Errorf("parseURL(%q).RequestURI() = %v, want %v", tt.s, u.RequestURI(), tt.rui)
		}
	}
}

var hostPortNoPortTests = []struct {
	u                    *url.URL
	hostPort, hostNoPort string
}{
	{&url.URL{Scheme: "ws", Host: "example.com"}, "example.com:80", "example.com"},
	{&url.URL{Scheme: "wss", Host: "example.com"}, "example.com:443", "example.com"},
	{&url.URL{Scheme: "ws", Host: "example.com:7777"}, "example.com:7777", "example.com"},
	{&url.URL{Scheme: "wss", Host: "example.com:7777"}, "example.com:7777", "example.com"},
}

func TestHostPortNoPort(t *testing.T) {
	for _, tt := range hostPortNoPortTests {
		hostPort, hostNoPort := hostPortNoPort(tt.u)
		if hostPort != tt.hostPort {
			t.Errorf("hostPortNoPort(%v) returned hostPort %q, want %q", tt.u, hostPort, tt.hostPort)
		}
		if hostNoPort != tt.hostNoPort {
			t.Errorf("hostPortNoPort(%v) returned hostNoPort %q, want %q", tt.u, hostNoPort, tt.hostNoPort)
		}
	}
}
