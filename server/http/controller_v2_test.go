// Copyright 2026 The frp Authors
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

package http

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/server/http/model"
	serverproxy "github.com/fatedier/frp/server/proxy"
	"github.com/fatedier/frp/server/registry"
)

type v2EnvelopeForTest[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type fakeStatsCollector struct {
	proxies map[string]*mem.ProxyStats
}

func (f *fakeStatsCollector) GetServer() *mem.ServerStats {
	return &mem.ServerStats{ProxyTypeCounts: map[string]int64{}}
}

func (f *fakeStatsCollector) GetProxiesByType(proxyType string) []*mem.ProxyStats {
	items := make([]*mem.ProxyStats, 0)
	for _, ps := range f.proxies {
		if ps.Type == proxyType {
			items = append(items, ps)
		}
	}
	return items
}

func (f *fakeStatsCollector) GetProxiesByTypeAndName(proxyType string, proxyName string) *mem.ProxyStats {
	ps := f.proxies[proxyName]
	if ps != nil && ps.Type == proxyType {
		return ps
	}
	return nil
}

func (f *fakeStatsCollector) GetProxyByName(proxyName string) *mem.ProxyStats {
	return f.proxies[proxyName]
}

func (f *fakeStatsCollector) GetProxyTraffic(name string) *mem.ProxyTrafficInfo {
	return nil
}

func (f *fakeStatsCollector) ClearOfflineProxies() (int, int) {
	return 0, len(f.proxies)
}

func TestAPIV2ClientListEnvelopePaginationAndFilters(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/clients?page=1&pageSize=1")
	if resp.Code != http.StatusOK {
		t.Fatalf("status mismatch, want %d got %d", http.StatusOK, resp.Code)
	}
	pageResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.ClientInfoResp]]](t, resp)
	if pageResp.Code != http.StatusOK || pageResp.Msg != "success" {
		t.Fatalf("envelope mismatch: %#v", pageResp)
	}
	if pageResp.Data.Total != 3 || pageResp.Data.Page != 1 || pageResp.Data.PageSize != 1 || len(pageResp.Data.Items) != 1 {
		t.Fatalf("page data mismatch: %#v", pageResp.Data)
	}
	if got := pageResp.Data.Items[0].User; got != "" {
		t.Fatalf("first sorted user mismatch, want empty got %q", got)
	}

	resp = performRequest(router, "/api/v2/clients?user=&page=1&pageSize=50")
	emptyUserResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.ClientInfoResp]]](t, resp)
	if emptyUserResp.Data.Total != 1 || emptyUserResp.Data.Items[0].User != "" {
		t.Fatalf("empty user filter mismatch: %#v", emptyUserResp.Data)
	}

	resp = performRequest(router, "/api/v2/clients?user=alice&status=online&q=alice-host")
	aliceResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.ClientInfoResp]]](t, resp)
	if aliceResp.Data.Total != 1 || aliceResp.Data.Items[0].User != "alice" {
		t.Fatalf("alice filter mismatch: %#v", aliceResp.Data)
	}

	resp = performRequest(router, "/api/v2/clients?status=offline")
	offlineResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.ClientInfoResp]]](t, resp)
	if offlineResp.Data.Total != 1 || offlineResp.Data.Items[0].User != "bob" {
		t.Fatalf("offline filter mismatch: %#v", offlineResp.Data)
	}
}

func TestAPIV2PageParamErrorsUseEnvelope(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/clients?page=0")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch, want %d got %d", http.StatusBadRequest, resp.Code)
	}
	errResp := decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusBadRequest || errResp.Data != nil {
		t.Fatalf("error envelope mismatch: %#v", errResp)
	}

	resp = performRequest(router, "/api/v2/clients?pageSize=201")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch, want %d got %d", http.StatusBadRequest, resp.Code)
	}

	resp = performRequest(router, fmt.Sprintf("/api/v2/clients?page=%d&pageSize=2", math.MaxInt))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch for overflowing page offset, want %d got %d", http.StatusBadRequest, resp.Code)
	}
}

func TestAPIV2ClientDetailEnvelope(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/clients/alice.client-a")
	if resp.Code != http.StatusOK {
		t.Fatalf("status mismatch, want %d got %d", http.StatusOK, resp.Code)
	}
	detailResp := decodeResponse[v2EnvelopeForTest[model.V2ClientDetailResp]](t, resp)
	if detailResp.Data.User != "alice" || detailResp.Data.ClientID != "client-a" {
		t.Fatalf("client detail mismatch: %#v", detailResp.Data)
	}
	if detailResp.Data.Status.State != "online" || detailResp.Data.Status.CurConns != 5 || detailResp.Data.Status.ProxyCount != 2 {
		t.Fatalf("client detail status mismatch: %#v", detailResp.Data.Status)
	}
}

func TestAPIV2ProxyListDetailAndUsers(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/proxies?type=invalid")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid proxy type status mismatch, want %d got %d", http.StatusBadRequest, resp.Code)
	}
	errResp := decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusBadRequest || errResp.Data != nil {
		t.Fatalf("invalid proxy type error envelope mismatch: %#v", errResp)
	}

	resp = performRequest(router, "/api/v2/proxies?type=tcp&user=&page=1&pageSize=50")
	proxyResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.V2ProxyResp]]](t, resp)
	if proxyResp.Data.Total != 1 {
		t.Fatalf("proxy filter total mismatch: %#v", proxyResp.Data)
	}
	proxyItem := proxyResp.Data.Items[0]
	if proxyItem.Name != "tcp-empty" || proxyItem.Type != "tcp" || proxyItem.User != "" || proxyItem.Status.State != "offline" {
		t.Fatalf("proxy item mismatch: %#v", proxyItem)
	}

	resp = performRequest(router, "/api/v2/proxies/tcp-alice")
	proxyDetailResp := decodeResponse[v2EnvelopeForTest[model.V2ProxyResp]](t, resp)
	if proxyDetailResp.Data.Name != "tcp-alice" || proxyDetailResp.Data.User != "alice" {
		t.Fatalf("proxy detail mismatch: %#v", proxyDetailResp.Data)
	}

	resp = performRequest(router, "/api/v2/users?page=1&pageSize=50")
	userResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[model.V2UserResp]]](t, resp)
	if userResp.Data.Total != 3 {
		t.Fatalf("user total mismatch: %#v", userResp.Data)
	}
	expectedProxyCounts := map[string]int{
		"":      1,
		"alice": 2,
		"bob":   1,
	}
	for _, item := range userResp.Data.Items {
		if item.ClientCount != 1 || item.ProxyCount != expectedProxyCounts[item.User] {
			t.Fatalf("user counts mismatch: %#v", item)
		}
	}
}

func TestMatchV2ProxyQueryMatchesSpecFields(t *testing.T) {
	tests := []struct {
		name string
		item model.V2ProxyResp
		q    string
		want bool
	}{
		{
			name: "tcp remote port",
			item: model.V2ProxyResp{Name: "tcp-proxy", Type: "tcp", Spec: &model.TCPOutConf{
				RemotePort: 6000,
			}},
			q:    "6000",
			want: true,
		},
		{
			name: "udp remote port",
			item: model.V2ProxyResp{Name: "udp-proxy", Type: "udp", Spec: &model.UDPOutConf{
				RemotePort: 7000,
			}},
			q:    "7000",
			want: true,
		},
		{
			name: "remote port does not match colon form",
			item: model.V2ProxyResp{Name: "tcp-proxy", Type: "tcp", Spec: &model.TCPOutConf{
				RemotePort: 6000,
			}},
			q:    ":6000",
			want: false,
		},
		{
			name: "http custom domain",
			item: model.V2ProxyResp{Name: "http-proxy", Type: "http", Spec: &model.HTTPOutConf{
				DomainConfig: v1.DomainConfig{CustomDomains: []string{"app.example.com"}},
			}},
			q:    "app.example.com",
			want: true,
		},
		{
			name: "https subdomain",
			item: model.V2ProxyResp{Name: "https-proxy", Type: "https", Spec: &model.HTTPSOutConf{
				DomainConfig: v1.DomainConfig{SubDomain: "portal"},
			}},
			q:    "portal",
			want: true,
		},
		{
			name: "subdomain does not match expanded host",
			item: model.V2ProxyResp{Name: "https-proxy", Type: "https", Spec: &model.HTTPSOutConf{
				DomainConfig: v1.DomainConfig{SubDomain: "portal"},
			}},
			q:    "portal.example.com",
			want: false,
		},
		{
			name: "tcpmux custom domain",
			item: model.V2ProxyResp{Name: "tcpmux-proxy", Type: "tcpmux", Spec: &model.TCPMuxOutConf{
				DomainConfig: v1.DomainConfig{CustomDomains: []string{"mux.example.com"}},
			}},
			q:    "mux.example.com",
			want: true,
		},
		{
			name: "nil spec does not match spec fields",
			item: model.V2ProxyResp{Name: "offline-proxy", Type: "tcp", Spec: nil},
			q:    "6000",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchV2ProxyQuery(tt.item, tt.q); got != tt.want {
				t.Fatalf("matchV2ProxyQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLegacyAPIResponsesRemainBare(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/clients")
	var clients []model.ClientInfoResp
	if err := json.Unmarshal(resp.Body.Bytes(), &clients); err != nil {
		t.Fatalf("legacy clients should be a bare array: %v, body: %s", err, resp.Body.String())
	}
	if len(clients) != 3 {
		t.Fatalf("legacy clients total mismatch, want 3 got %d", len(clients))
	}

	resp = performRequest(router, "/api/proxy/tcp")
	var proxies model.GetProxyInfoResp
	if err := json.Unmarshal(resp.Body.Bytes(), &proxies); err != nil {
		t.Fatalf("legacy proxy response should be {proxies}: %v, body: %s", err, resp.Body.String())
	}
	if len(proxies.Proxies) != 2 {
		t.Fatalf("legacy tcp proxy total mismatch, want 2 got %d", len(proxies.Proxies))
	}
	var envelope httppkg.V2Response
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err == nil && envelope.Code != 0 {
		t.Fatalf("legacy proxy response should not use v2 envelope: %#v", envelope)
	}
}

func newV2TestController(t *testing.T) *Controller {
	t.Helper()

	oldStatsCollector := mem.StatsCollector
	mem.StatsCollector = &fakeStatsCollector{
		proxies: map[string]*mem.ProxyStats{
			"tcp-empty": {
				Name:            "tcp-empty",
				Type:            "tcp",
				User:            "",
				ClientID:        "legacy-client",
				TodayTrafficIn:  10,
				TodayTrafficOut: 20,
				CurConns:        1,
			},
			"tcp-alice": {
				Name:            "tcp-alice",
				Type:            "tcp",
				User:            "alice",
				ClientID:        "client-a",
				TodayTrafficIn:  30,
				TodayTrafficOut: 40,
				CurConns:        2,
			},
			"http-alice": {
				Name:     "http-alice",
				Type:     "http",
				User:     "alice",
				ClientID: "client-a",
				CurConns: 3,
			},
			"udp-bob": {
				Name:     "udp-bob",
				Type:     "udp",
				User:     "bob",
				ClientID: "client-b",
			},
		},
	}
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	clientRegistry := registry.NewClientRegistry()
	clientRegistry.Register("", "legacy-client", "run-empty", "empty-host", "1.0.0", "127.0.0.1", "v1")
	clientRegistry.Register("alice", "client-a", "run-a", "alice-host", "1.0.0", "127.0.0.2", "v2")
	clientRegistry.Register("bob", "client-b", "run-b", "bob-host", "1.0.0", "127.0.0.3", "v1")
	clientRegistry.MarkOfflineByRunID("run-b")

	return NewController(&v1.ServerConfig{}, clientRegistry, serverproxy.NewManager())
}

func newV2TestRouter(controller *Controller) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/api/v2/users", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2UserList)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/clients", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ClientList)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/clients/{key}", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ClientDetail)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/proxies", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyList)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/proxies/{name}", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyDetail)).Methods(http.MethodGet)
	router.HandleFunc("/api/clients", httppkg.MakeHTTPHandlerFunc(controller.APIClientList)).Methods(http.MethodGet)
	router.HandleFunc("/api/proxy/{type}", httppkg.MakeHTTPHandlerFunc(controller.APIProxyByType)).Methods(http.MethodGet)
	return router
}

func performRequest(handler http.Handler, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func decodeResponse[T any](t *testing.T, resp *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response failed: %v, body: %s", err, resp.Body.String())
	}
	return out
}
