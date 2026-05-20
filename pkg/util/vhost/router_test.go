// Copyright 2017 fatedier, fatedier@gmail.com
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

package vhost

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRoutersGet(t *testing.T) {
	routers := NewRouters()
	require.NoError(t, routers.Add("example.com", "/api", "alice", "exact-user"))
	require.NoError(t, routers.Add("example.com", "/public", "", "exact-all-users"))
	require.NoError(t, routers.Add("*.example.com", "/api", "", "wildcard-all-users"))
	require.NoError(t, routers.Add("*", "/api", "", "all-domains"))

	t.Run("exact host and user match with normalized domain", func(t *testing.T) {
		router, ok := routers.Get("EXAMPLE.COM", "/api/users", "alice")
		require.True(t, ok)
		require.Equal(t, "exact-user", router.payload)
	})

	t.Run("exact host and empty user match", func(t *testing.T) {
		router, ok := routers.Get("EXAMPLE.COM", "/public/docs", "")
		require.True(t, ok)
		require.Equal(t, "exact-all-users", router.payload)
	})

	t.Run("does not fall back from named user to empty user", func(t *testing.T) {
		// Get intentionally requires an exact HTTP user; route-level fallbacks live in getByRoute.
		_, ok := routers.Get("EXAMPLE.COM", "/public/docs", "alice")
		require.False(t, ok)
	})

	t.Run("does not fall back to wildcard domains", func(t *testing.T) {
		_, ok := routers.Get("foo.example.com", "/api/users", "")
		require.False(t, ok)
	})

	t.Run("does not fall back to catch-all domain", func(t *testing.T) {
		_, ok := routers.Get("missing.test", "/api/users", "")
		require.False(t, ok)
	})
}

func TestRoutersGetByRoute(t *testing.T) {
	routers := NewRouters()
	require.NoError(t, routers.Add("example.com", "/api", "alice", "exact-user"))
	require.NoError(t, routers.Add("example.com", "/api", "", "exact-all-users"))
	require.NoError(t, routers.Add("exact.example.com", "/api", "", "exact-subdomain"))
	require.NoError(t, routers.Add("*.example.com", "/api", "", "wildcard-all-users"))
	require.NoError(t, routers.Add("*.foo.example.com", "/api", "", "specific-wildcard"))
	require.NoError(t, routers.Add("*.bar.com", "/api", "", "wildcard-parent-domain"))
	require.NoError(t, routers.Add("*", "/admin", "root", "all-domains-user"))
	require.NoError(t, routers.Add("*", "/", "", "all-domains"))

	tests := []struct {
		name     string
		domain   string
		location string
		httpUser string
		want     string
	}{
		{
			name:     "exact domain and http user",
			domain:   "example.com",
			location: "/api/users",
			httpUser: "alice",
			want:     "exact-user",
		},
		{
			name:     "exact domain falls back to all users",
			domain:   "example.com",
			location: "/api/users",
			httpUser: "bob",
			want:     "exact-all-users",
		},
		{
			name:     "wildcard domain uses all users fallback",
			domain:   "foo.example.com",
			location: "/api/users",
			httpUser: "bob",
			want:     "wildcard-all-users",
		},
		{
			name:     "mixed-case domain is normalized",
			domain:   "Foo.Example.Com",
			location: "/api/users",
			httpUser: "bob",
			want:     "wildcard-all-users",
		},
		{
			name:     "exact domain wins over wildcard domain",
			domain:   "exact.example.com",
			location: "/api/users",
			httpUser: "bob",
			want:     "exact-subdomain",
		},
		{
			name:     "more specific wildcard wins over broader wildcard",
			domain:   "bar.foo.example.com",
			location: "/api/users",
			httpUser: "bob",
			want:     "specific-wildcard",
		},
		{
			name:     "wildcard walk checks parent domains",
			domain:   "a.b.bar.com",
			location: "/api/users",
			httpUser: "bob",
			want:     "wildcard-parent-domain",
		},
		{
			name:     "catch-all domain fallback",
			domain:   "foo.test.com",
			location: "/other",
			httpUser: "bob",
			want:     "all-domains",
		},
		{
			name:     "catch-all domain honors http user",
			domain:   "foo.test.com",
			location: "/admin/panel",
			httpUser: "root",
			want:     "all-domains-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, ok := routers.getByRoute(tt.domain, tt.location, tt.httpUser)
			require.True(t, ok)
			require.Equal(t, tt.want, router.payload)
		})
	}
}

func TestRoutersGetByRouteNoMatch(t *testing.T) {
	routers := NewRouters()
	require.NoError(t, routers.Add("*.example.com", "/api", "", "wildcard-all-users"))
	require.NoError(t, routers.Add("*.com", "/api", "", "top-level-wildcard"))

	tests := []struct {
		name     string
		domain   string
		location string
	}{
		{
			name:     "two-label domain does not enter wildcard walk",
			domain:   "example.com",
			location: "/api/users",
		},
		{
			name:     "missing catch-all remains no match",
			domain:   "foo.test.com",
			location: "/api/users",
		},
		{
			name:     "wrong path remains no match",
			domain:   "foo.example.com",
			location: "/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, ok := routers.getByRoute(tt.domain, tt.location, "bob")
			require.False(t, ok)
			require.Nil(t, router)
		})
	}
}

func TestRoutersConcurrentGetByRouteAndAdd(t *testing.T) {
	routers := NewRouters()
	require.NoError(t, routers.Add("*.example.com", "/api", "", "wildcard"))

	const readers = 8
	const iterations = 200

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := make(chan struct{})
	errCh := make(chan error, readers+1)
	var wg sync.WaitGroup

	for id := range readers {
		wg.Go(func() {
			<-start
			for j := range iterations {
				select {
				case <-ctx.Done():
					return
				default:
				}
				router, ok := routers.getByRoute("foo.example.com", "/api/users", "")
				if !ok || router == nil || router.payload != "wildcard" {
					errCh <- fmt.Errorf("reader %d iteration %d got router=%v ok=%v", id, j, router, ok)
					return
				}
			}
		})
	}

	wg.Go(func() {
		<-start
		for i := range iterations {
			select {
			case <-ctx.Done():
				return
			default:
			}
			err := routers.Add(fmt.Sprintf("host-%d.example.com", i), "/api", "", i)
			if err != nil {
				errCh <- err
				return
			}
		}
	})

	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("concurrent route lookup and add timed out")
	}

	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
}
