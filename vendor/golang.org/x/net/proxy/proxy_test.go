// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proxy

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/internal/sockstest"
)

type proxyFromEnvTest struct {
	allProxyEnv string
	noProxyEnv  string
	wantTypeOf  Dialer
}

func (t proxyFromEnvTest) String() string {
	var buf bytes.Buffer
	space := func() {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
	}
	if t.allProxyEnv != "" {
		fmt.Fprintf(&buf, "all_proxy=%q", t.allProxyEnv)
	}
	if t.noProxyEnv != "" {
		space()
		fmt.Fprintf(&buf, "no_proxy=%q", t.noProxyEnv)
	}
	return strings.TrimSpace(buf.String())
}

func TestFromEnvironment(t *testing.T) {
	ResetProxyEnv()

	type dummyDialer struct {
		direct
	}

	RegisterDialerType("irc", func(_ *url.URL, _ Dialer) (Dialer, error) {
		return dummyDialer{}, nil
	})

	proxyFromEnvTests := []proxyFromEnvTest{
		{allProxyEnv: "127.0.0.1:8080", noProxyEnv: "localhost, 127.0.0.1", wantTypeOf: direct{}},
		{allProxyEnv: "ftp://example.com:8000", noProxyEnv: "localhost, 127.0.0.1", wantTypeOf: direct{}},
		{allProxyEnv: "socks5://example.com:8080", noProxyEnv: "localhost, 127.0.0.1", wantTypeOf: &PerHost{}},
		{allProxyEnv: "irc://example.com:8000", wantTypeOf: dummyDialer{}},
		{noProxyEnv: "localhost, 127.0.0.1", wantTypeOf: direct{}},
		{wantTypeOf: direct{}},
	}

	for _, tt := range proxyFromEnvTests {
		os.Setenv("ALL_PROXY", tt.allProxyEnv)
		os.Setenv("NO_PROXY", tt.noProxyEnv)
		ResetCachedEnvironment()

		d := FromEnvironment()
		if got, want := fmt.Sprintf("%T", d), fmt.Sprintf("%T", tt.wantTypeOf); got != want {
			t.Errorf("%v: got type = %T, want %T", tt, d, tt.wantTypeOf)
		}
	}
}

func TestFromURL(t *testing.T) {
	ss, err := sockstest.NewServer(sockstest.NoAuthRequired, sockstest.NoProxyRequired)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	url, err := url.Parse("socks5://user:password@" + ss.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := FromURL(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := proxy.Dial("tcp", "fqdn.doesnotexist:5963")
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestSOCKS5(t *testing.T) {
	ss, err := sockstest.NewServer(sockstest.NoAuthRequired, sockstest.NoProxyRequired)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	proxy, err := SOCKS5("tcp", ss.Addr().String(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := proxy.Dial("tcp", ss.TargetAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func ResetProxyEnv() {
	for _, env := range []*envOnce{allProxyEnv, noProxyEnv} {
		for _, v := range env.names {
			os.Setenv(v, "")
		}
	}
	ResetCachedEnvironment()
}

func ResetCachedEnvironment() {
	allProxyEnv.reset()
	noProxyEnv.reset()
}
