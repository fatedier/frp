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
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/fatedier/frp/pkg/config/types"
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
	server    *mem.ServerStats
	proxies   map[string]*mem.ProxyStats
	traffic   map[string]*mem.ProxyTrafficInfo
	pruneable map[string]bool
}

func (f *fakeStatsCollector) GetServer() *mem.ServerStats {
	if f.server != nil {
		return f.server
	}
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
	return f.traffic[name]
}

func (f *fakeStatsCollector) ClearOfflineProxies() (int, int) {
	return 0, len(f.proxies)
}

func (f *fakeStatsCollector) PruneOfflineProxies() (int, int) {
	total := len(f.proxies)
	cleared := 0
	for name := range f.pruneable {
		if _, ok := f.proxies[name]; ok {
			delete(f.proxies, name)
			cleared++
		}
	}
	f.pruneable = map[string]bool{}
	return cleared, total
}

func TestAPIV2SystemInfoEnvelope(t *testing.T) {
	oldStatsCollector := mem.StatsCollector
	mem.StatsCollector = &fakeStatsCollector{
		server: &mem.ServerStats{
			TotalTrafficIn:  1024,
			TotalTrafficOut: 2048,
			CurConns:        3,
			ClientCounts:    4,
			ProxyTypeCounts: map[string]int64{
				"tcp":  2,
				"http": 1,
			},
		},
		proxies: map[string]*mem.ProxyStats{},
	}
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	controller := NewController(&v1.ServerConfig{
		BindPort:              7000,
		VhostHTTPPort:         8080,
		VhostHTTPSPort:        8443,
		TCPMuxHTTPConnectPort: 9000,
		KCPBindPort:           7001,
		QUICBindPort:          7002,
		SubDomainHost:         "example.com",
		MaxPortsPerClient:     8,
		AllowPorts: []types.PortsRange{
			{Start: 1000, End: 1002},
			{Single: 2000},
		},
		Transport: v1.ServerTransportConfig{
			MaxPoolCount:     5,
			HeartbeatTimeout: 90,
			TLS: v1.TLSServerConfig{
				Force: true,
			},
		},
	}, registry.NewClientRegistry(), serverproxy.NewManager())
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/system/info")
	if resp.Code != http.StatusOK {
		t.Fatalf("status mismatch, want %d got %d", http.StatusOK, resp.Code)
	}

	rawResp := decodeResponse[v2EnvelopeForTest[map[string]json.RawMessage]](t, resp)
	if rawResp.Code != http.StatusOK || rawResp.Msg != "success" {
		t.Fatalf("envelope mismatch: %#v", rawResp)
	}
	assertRawJSONKeys(t, rawResp.Data, "config", "status", "version")
	assertRawJSONKeysFromMessage(t, rawResp.Data["config"],
		"allowPortsStr",
		"bindPort",
		"heartbeatTimeout",
		"kcpBindPort",
		"maxPoolCount",
		"maxPortsPerClient",
		"quicBindPort",
		"subdomainHost",
		"tcpmuxHTTPConnectPort",
		"tlsForce",
		"vhostHTTPPort",
		"vhostHTTPSPort",
	)
	assertRawJSONKeysFromMessage(t, rawResp.Data["status"],
		"clientCounts",
		"curConns",
		"proxyTypeCount",
		"totalTrafficIn",
		"totalTrafficOut",
	)

	systemResp := decodeResponse[v2EnvelopeForTest[model.V2SystemInfoResp]](t, resp)
	if systemResp.Data.Version == "" {
		t.Fatal("version should be set at top level")
	}
	if systemResp.Data.Config.BindPort != 7000 ||
		systemResp.Data.Config.VhostHTTPPort != 8080 ||
		systemResp.Data.Config.VhostHTTPSPort != 8443 ||
		systemResp.Data.Config.TCPMuxHTTPConnectPort != 9000 ||
		systemResp.Data.Config.KCPBindPort != 7001 ||
		systemResp.Data.Config.QUICBindPort != 7002 ||
		systemResp.Data.Config.SubdomainHost != "example.com" ||
		systemResp.Data.Config.MaxPoolCount != 5 ||
		systemResp.Data.Config.MaxPortsPerClient != 8 ||
		systemResp.Data.Config.HeartbeatTimeout != 90 ||
		systemResp.Data.Config.AllowPortsStr != "1000-1002,2000" ||
		!systemResp.Data.Config.TLSForce {
		t.Fatalf("config mismatch: %#v", systemResp.Data.Config)
	}
	if systemResp.Data.Status.TotalTrafficIn != 1024 ||
		systemResp.Data.Status.TotalTrafficOut != 2048 ||
		systemResp.Data.Status.CurConns != 3 ||
		systemResp.Data.Status.ClientCounts != 4 ||
		systemResp.Data.Status.ProxyTypeCounts["tcp"] != 2 ||
		systemResp.Data.Status.ProxyTypeCounts["http"] != 1 {
		t.Fatalf("status mismatch: %#v", systemResp.Data.Status)
	}
}

func TestAPIV2SystemPruneOfflineProxies(t *testing.T) {
	oldStatsCollector := mem.StatsCollector
	collector := &fakeStatsCollector{
		proxies: map[string]*mem.ProxyStats{
			"tcp-offline":    {Name: "tcp-offline", Type: "tcp"},
			"http-offline":   {Name: "http-offline", Type: "http"},
			"udp-offline":    {Name: "udp-offline", Type: "udp"},
			"tcp-online":     {Name: "tcp-online", Type: "tcp"},
			"http-online":    {Name: "http-online", Type: "http"},
			"udp-online":     {Name: "udp-online", Type: "udp"},
			"stcp-restarted": {Name: "stcp-restarted", Type: "stcp"},
			"xtcp-restarted": {Name: "xtcp-restarted", Type: "xtcp"},
			"sudp-same-time": {Name: "sudp-same-time", Type: "sudp"},
			"tcpmux-running": {Name: "tcpmux-running", Type: "tcpmux"},
		},
		pruneable: map[string]bool{
			"tcp-offline":  true,
			"http-offline": true,
			"udp-offline":  true,
		},
	}
	mem.StatsCollector = collector
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	controller := NewController(&v1.ServerConfig{}, registry.NewClientRegistry(), serverproxy.NewManager())
	router := newV2TestRouter(controller)

	resp := performRequestWithMethod(router, http.MethodPost, "/api/v2/system/prune?type=offline_proxies")
	if resp.Code != http.StatusOK {
		t.Fatalf("status mismatch, want %d got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	rawResp := decodeResponse[v2EnvelopeForTest[map[string]json.RawMessage]](t, resp)
	if rawResp.Code != http.StatusOK || rawResp.Msg != "success" {
		t.Fatalf("envelope mismatch: %#v", rawResp)
	}
	assertRawJSONKeys(t, rawResp.Data, "cleared", "total", "type")
	pruneResp := decodeResponse[v2EnvelopeForTest[model.V2SystemPruneResp]](t, resp)
	if pruneResp.Data.Type != "offline_proxies" || pruneResp.Data.Cleared != 3 || pruneResp.Data.Total != 10 {
		t.Fatalf("prune response mismatch: %#v", pruneResp.Data)
	}
	if _, ok := collector.proxies["tcp-offline"]; ok {
		t.Fatal("pruned proxy statistics should be removed")
	}
	if _, ok := collector.proxies["tcp-online"]; !ok {
		t.Fatal("online proxy statistics should remain")
	}

	resp = performRequestWithMethod(router, http.MethodPost, "/api/v2/system/prune?type=offline_proxies")
	if resp.Code != http.StatusOK {
		t.Fatalf("second prune status mismatch, want %d got %d", http.StatusOK, resp.Code)
	}
	pruneResp = decodeResponse[v2EnvelopeForTest[model.V2SystemPruneResp]](t, resp)
	if pruneResp.Data.Cleared != 0 || pruneResp.Data.Total != 7 {
		t.Fatalf("second prune response mismatch: %#v", pruneResp.Data)
	}
}

func TestAPIV2SystemPruneTypeErrorsUseEnvelope(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequestWithMethod(router, http.MethodPost, "/api/v2/system/prune")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("missing type status mismatch, want %d got %d", http.StatusBadRequest, resp.Code)
	}
	errResp := decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusBadRequest || errResp.Msg != "type is required" || errResp.Data != nil {
		t.Fatalf("missing type error envelope mismatch: %#v", errResp)
	}

	resp = performRequestWithMethod(router, http.MethodPost, "/api/v2/system/prune?type=clients")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid type status mismatch, want %d got %d", http.StatusBadRequest, resp.Code)
	}
	errResp = decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusBadRequest || errResp.Msg != "type must be one of offline_proxies" || errResp.Data != nil {
		t.Fatalf("invalid type error envelope mismatch: %#v", errResp)
	}
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

func TestAPIV2ClientDetailEncodedKey(t *testing.T) {
	oldStatsCollector := mem.StatsCollector
	mem.StatsCollector = &fakeStatsCollector{
		proxies: map[string]*mem.ProxyStats{
			"tcp-url": {
				Name:     "tcp-url",
				Type:     "tcp",
				User:     "url",
				ClientID: "client/a?b#c",
				CurConns: 7,
			},
		},
	}
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	clientRegistry := registry.NewClientRegistry()
	clientRegistry.Register("url", "client/a?b#c", "run-url", "url-host", "1.0.0", "127.0.0.4", "v2")
	controller := NewController(&v1.ServerConfig{}, clientRegistry, serverproxy.NewManager())
	router := newV2TestRouter(controller)

	encodedKey := url.PathEscape("url.client/a?b#c")
	resp := performRequest(router, "/api/v2/clients/"+encodedKey)
	if resp.Code != http.StatusOK {
		t.Fatalf("encoded client key status mismatch, want %d got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	encodedResp := decodeResponse[v2EnvelopeForTest[model.V2ClientDetailResp]](t, resp)
	if encodedResp.Data.User != "url" || encodedResp.Data.ClientID != "client/a?b#c" {
		t.Fatalf("encoded client detail mismatch: %#v", encodedResp.Data)
	}
	if encodedResp.Data.Status.CurConns != 7 || encodedResp.Data.Status.ProxyCount != 1 {
		t.Fatalf("encoded client detail status mismatch: %#v", encodedResp.Data.Status)
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
	if proxyItem.Name != "tcp-empty" || proxyItem.Spec.Type != "tcp" || proxyItem.User != "" || proxyItem.Status.State != "offline" {
		t.Fatalf("proxy item mismatch: %#v", proxyItem)
	}
	rawProxyResp := decodeResponse[v2EnvelopeForTest[model.V2PageResp[map[string]json.RawMessage]]](t, resp)
	assertRawJSONKeys(t, rawProxyResp.Data.Items[0], "clientID", "name", "spec", "status", "user")
	var rawListSpec map[string]json.RawMessage
	if err := json.Unmarshal(rawProxyResp.Data.Items[0]["spec"], &rawListSpec); err != nil {
		t.Fatalf("unmarshal list proxy spec failed: %v", err)
	}
	assertRawJSONKeys(t, rawListSpec, "tcp", "type")
	assertRawJSONKeysFromMessage(t, rawListSpec["tcp"])

	resp = performRequest(router, "/api/v2/proxies/tcp-alice")
	rawProxyDetailResp := decodeResponse[v2EnvelopeForTest[map[string]json.RawMessage]](t, resp)
	assertRawJSONKeysFromMessage(t, rawProxyDetailResp.Data["status"],
		"curConns",
		"lastCloseAt",
		"lastStartAt",
		"phase",
		"todayTrafficIn",
		"todayTrafficOut",
	)
	proxyDetailResp := decodeResponse[v2EnvelopeForTest[model.V2ProxyResp]](t, resp)
	if proxyDetailResp.Data.Name != "tcp-alice" || proxyDetailResp.Data.User != "alice" {
		t.Fatalf("proxy detail mismatch: %#v", proxyDetailResp.Data)
	}
	assertRawJSONKeys(t, rawProxyDetailResp.Data, "clientID", "name", "spec", "status", "user")
	var rawDetailSpec map[string]json.RawMessage
	if err := json.Unmarshal(rawProxyDetailResp.Data["spec"], &rawDetailSpec); err != nil {
		t.Fatalf("unmarshal detail proxy spec failed: %v", err)
	}
	assertRawJSONKeys(t, rawDetailSpec, "tcp", "type")
	assertRawJSONKeysFromMessage(t, rawDetailSpec["tcp"])
	if proxyDetailResp.Data.Status.LastStartAt != 1783504200 || proxyDetailResp.Data.Status.LastCloseAt != 1783504300 {
		t.Fatalf("proxy detail timestamp mismatch: %#v", proxyDetailResp.Data.Status)
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

func TestAPIV2ProxyTrafficEnvelopeSchemaAndHistory(t *testing.T) {
	oldStatsCollector := mem.StatsCollector
	mem.StatsCollector = &fakeStatsCollector{
		proxies: map[string]*mem.ProxyStats{
			"ssh": {Name: "ssh", Type: "tcp"},
		},
		traffic: map[string]*mem.ProxyTrafficInfo{
			"ssh": {
				Name:       "ssh",
				TrafficIn:  []int64{70, 60, 50, 40, 30, 20, 10},
				TrafficOut: []int64{700, 600, 500, 400, 300, 200, 100},
			},
		},
	}
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	controller := NewController(&v1.ServerConfig{}, registry.NewClientRegistry(), serverproxy.NewManager())
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/proxies/ssh/traffic")
	if resp.Code != http.StatusOK {
		t.Fatalf("status mismatch, want %d got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	rawResp := decodeResponse[v2EnvelopeForTest[map[string]json.RawMessage]](t, resp)
	if rawResp.Code != http.StatusOK || rawResp.Msg != "success" {
		t.Fatalf("envelope mismatch: %#v", rawResp)
	}
	assertRawJSONKeys(t, rawResp.Data, "granularity", "history", "name", "unit")

	trafficResp := decodeResponse[v2EnvelopeForTest[model.V2ProxyTrafficResp]](t, resp)
	if trafficResp.Data.Name != "ssh" || trafficResp.Data.Unit != "bytes" || trafficResp.Data.Granularity != "day" {
		t.Fatalf("traffic metadata mismatch: %#v", trafficResp.Data)
	}
	if len(trafficResp.Data.History) != 7 {
		t.Fatalf("history length mismatch, want 7 got %d: %#v", len(trafficResp.Data.History), trafficResp.Data.History)
	}

	wantIn := []int64{10, 20, 30, 40, 50, 60, 70}
	wantOut := []int64{100, 200, 300, 400, 500, 600, 700}
	var prevDate time.Time
	for i, point := range trafficResp.Data.History {
		assertRawJSONKeysFromMessage(t, mustMarshalJSON(t, point), "date", "trafficIn", "trafficOut")
		if point.TrafficIn != wantIn[i] || point.TrafficOut != wantOut[i] {
			t.Fatalf("history[%d] traffic mismatch: %#v", i, point)
		}
		parsedDate, err := time.Parse(time.DateOnly, point.Date)
		if err != nil {
			t.Fatalf("history[%d] date should be yyyy-mm-dd, got %q: %v", i, point.Date, err)
		}
		if i > 0 && !parsedDate.Equal(prevDate.AddDate(0, 0, 1)) {
			t.Fatalf("history dates should be oldest to newest, got %s after %s", point.Date, prevDate.Format(time.DateOnly))
		}
		prevDate = parsedDate
	}
}

func TestAPIV2ProxyTrafficNotFoundEnvelope(t *testing.T) {
	controller := newV2TestController(t)
	router := newV2TestRouter(controller)

	resp := performRequest(router, "/api/v2/proxies/missing/traffic")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status mismatch, want %d got %d, body: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
	errResp := decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusNotFound || errResp.Msg != "no proxy info found" || errResp.Data != nil {
		t.Fatalf("not found envelope mismatch: %#v", errResp)
	}
}

func TestAPIV2ProxyDetailAndTrafficEncodedName(t *testing.T) {
	name := "folder/ssh?x#y"
	oldStatsCollector := mem.StatsCollector
	mem.StatsCollector = &fakeStatsCollector{
		proxies: map[string]*mem.ProxyStats{
			name: {Name: name, Type: "tcp", User: "encoded"},
		},
		traffic: map[string]*mem.ProxyTrafficInfo{
			name: {
				Name:       name,
				TrafficIn:  []int64{1},
				TrafficOut: []int64{2},
			},
		},
	}
	t.Cleanup(func() {
		mem.StatsCollector = oldStatsCollector
	})

	controller := NewController(&v1.ServerConfig{}, registry.NewClientRegistry(), serverproxy.NewManager())
	router := newV2TestRouter(controller)
	encodedName := url.PathEscape(name)

	resp := performRequest(router, "/api/v2/proxies/"+encodedName)
	if resp.Code != http.StatusOK {
		t.Fatalf("encoded proxy detail status mismatch, want %d got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	detailResp := decodeResponse[v2EnvelopeForTest[model.V2ProxyResp]](t, resp)
	if detailResp.Data.Name != name || detailResp.Data.User != "encoded" {
		t.Fatalf("encoded proxy detail mismatch: %#v", detailResp.Data)
	}

	resp = performRequest(router, "/api/v2/proxies/"+encodedName+"/traffic")
	if resp.Code != http.StatusOK {
		t.Fatalf("encoded traffic status mismatch, want %d got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	trafficResp := decodeResponse[v2EnvelopeForTest[model.V2ProxyTrafficResp]](t, resp)
	if trafficResp.Data.Name != name {
		t.Fatalf("encoded traffic name mismatch: %#v", trafficResp.Data)
	}
	if got := trafficResp.Data.History[len(trafficResp.Data.History)-1]; got.TrafficIn != 1 || got.TrafficOut != 2 {
		t.Fatalf("encoded traffic latest point mismatch: %#v", got)
	}
}

func TestAPIV2ProxyTrafficInvalidEncodedNameUses400Envelope(t *testing.T) {
	controller := newV2TestController(t)
	handler := httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyTraffic)
	req := httptest.NewRequest(http.MethodGet, "/api/v2/proxies/%25ZZ/traffic", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "%ZZ"})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch, want %d got %d, body: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}
	errResp := decodeResponse[httppkg.V2Response](t, resp)
	if errResp.Code != http.StatusBadRequest || errResp.Msg != "invalid proxy name" || errResp.Data != nil {
		t.Fatalf("invalid encoded name envelope mismatch: %#v", errResp)
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
			item: model.V2ProxyResp{Name: "tcp-proxy", Spec: model.V2ProxySpec{
				Type: "tcp",
				TCP:  &model.V2TCPProxySpec{RemotePort: v2TestIntPtr(6000)},
			}},
			q:    "6000",
			want: true,
		},
		{
			name: "udp remote port",
			item: model.V2ProxyResp{Name: "udp-proxy", Spec: model.V2ProxySpec{
				Type: "udp",
				UDP:  &model.V2UDPProxySpec{RemotePort: v2TestIntPtr(7000)},
			}},
			q:    "7000",
			want: true,
		},
		{
			name: "remote port does not match colon form",
			item: model.V2ProxyResp{Name: "tcp-proxy", Spec: model.V2ProxySpec{
				Type: "tcp",
				TCP:  &model.V2TCPProxySpec{RemotePort: v2TestIntPtr(6000)},
			}},
			q:    ":6000",
			want: false,
		},
		{
			name: "http custom domain",
			item: model.V2ProxyResp{Name: "http-proxy", Spec: model.V2ProxySpec{
				Type: "http",
				HTTP: &model.V2HTTPProxySpec{CustomDomains: []string{"app.example.com"}},
			}},
			q:    "app.example.com",
			want: true,
		},
		{
			name: "https subdomain",
			item: model.V2ProxyResp{Name: "https-proxy", Spec: model.V2ProxySpec{
				Type:  "https",
				HTTPS: &model.V2HTTPSProxySpec{Subdomain: "portal"},
			}},
			q:    "portal",
			want: true,
		},
		{
			name: "subdomain does not match expanded host",
			item: model.V2ProxyResp{Name: "https-proxy", Spec: model.V2ProxySpec{
				Type:  "https",
				HTTPS: &model.V2HTTPSProxySpec{Subdomain: "portal"},
			}},
			q:    "portal.example.com",
			want: false,
		},
		{
			name: "tcpmux custom domain",
			item: model.V2ProxyResp{Name: "tcpmux-proxy", Spec: model.V2ProxySpec{
				Type:   "tcpmux",
				TCPMux: &model.V2TCPMuxProxySpec{CustomDomains: []string{"mux.example.com"}},
			}},
			q:    "mux.example.com",
			want: true,
		},
		{
			name: "offline shell does not match online spec fields",
			item: model.V2ProxyResp{Name: "offline-proxy", Spec: model.V2ProxySpec{
				Type: "tcp",
				TCP:  &model.V2TCPProxySpec{},
			}},
			q:    "6000",
			want: false,
		},
		{
			name: "offline shell does not contribute zero remote port",
			item: model.V2ProxyResp{Name: "offline-proxy", Spec: model.V2ProxySpec{
				Type: "tcp",
				TCP:  &model.V2TCPProxySpec{},
			}},
			q:    "0",
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

	resp := performRequest(router, "/api/serverinfo")
	var serverInfo model.ServerInfoResp
	if err := json.Unmarshal(resp.Body.Bytes(), &serverInfo); err != nil {
		t.Fatalf("legacy serverinfo should be a bare object: %v, body: %s", err, resp.Body.String())
	}
	if serverInfo.Version == "" {
		t.Fatal("legacy serverinfo version should be set")
	}
	var serverInfoRaw map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body.Bytes(), &serverInfoRaw); err != nil {
		t.Fatalf("unmarshal legacy serverinfo object failed: %v", err)
	}
	if _, ok := serverInfoRaw["data"]; ok {
		t.Fatalf("legacy serverinfo should not use v2 envelope: %s", resp.Body.String())
	}
	if _, ok := serverInfoRaw["config"]; ok {
		t.Fatalf("legacy serverinfo should stay flat, got config in: %s", resp.Body.String())
	}

	resp = performRequest(router, "/api/clients")
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

	resp = performRequest(router, "/api/traffic/tcp-alice")
	var traffic model.GetProxyTrafficResp
	if err := json.Unmarshal(resp.Body.Bytes(), &traffic); err != nil {
		t.Fatalf("legacy traffic should be a bare object: %v, body: %s", err, resp.Body.String())
	}
	if traffic.Name != "tcp-alice" ||
		len(traffic.TrafficIn) != 2 || traffic.TrafficIn[0] != 7 || traffic.TrafficIn[1] != 6 ||
		len(traffic.TrafficOut) != 2 || traffic.TrafficOut[0] != 70 || traffic.TrafficOut[1] != 60 {
		t.Fatalf("legacy traffic should preserve today-first arrays, got: %#v", traffic)
	}
	var trafficRaw map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body.Bytes(), &trafficRaw); err != nil {
		t.Fatalf("unmarshal legacy traffic object failed: %v", err)
	}
	if _, ok := trafficRaw["data"]; ok {
		t.Fatalf("legacy traffic should not use v2 envelope: %s", resp.Body.String())
	}
}

func v2TestIntPtr(value int) *int {
	return &value
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
				LastStartTime:   "07-08 12:30:00",
				LastCloseTime:   "07-08 12:31:40",
				LastStartAt:     1783504200,
				LastCloseAt:     1783504300,
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
		traffic: map[string]*mem.ProxyTrafficInfo{
			"tcp-alice": {
				Name:       "tcp-alice",
				TrafficIn:  []int64{7, 6},
				TrafficOut: []int64{70, 60},
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
	router.HandleFunc("/api/v2/system/info", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2SystemInfo)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/system/prune", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2SystemPrune)).Methods(http.MethodPost)
	router.HandleFunc("/api/v2/clients", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ClientList)).Methods(http.MethodGet)
	encodedPathRouter := router.NewRoute().Subrouter()
	encodedPathRouter.UseEncodedPath()
	encodedPathRouter.HandleFunc("/api/v2/clients/{key}", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ClientDetail)).Methods(http.MethodGet)
	router.HandleFunc("/api/v2/proxies", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyList)).Methods(http.MethodGet)
	encodedPathRouter.HandleFunc("/api/v2/proxies/{name}/traffic", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyTraffic)).Methods(http.MethodGet)
	encodedPathRouter.HandleFunc("/api/v2/proxies/{name}", httppkg.MakeHTTPHandlerFuncV2(controller.APIV2ProxyDetail)).Methods(http.MethodGet)
	router.HandleFunc("/api/serverinfo", httppkg.MakeHTTPHandlerFunc(controller.APIServerInfo)).Methods(http.MethodGet)
	router.HandleFunc("/api/clients", httppkg.MakeHTTPHandlerFunc(controller.APIClientList)).Methods(http.MethodGet)
	router.HandleFunc("/api/proxy/{type}", httppkg.MakeHTTPHandlerFunc(controller.APIProxyByType)).Methods(http.MethodGet)
	router.HandleFunc("/api/traffic/{name}", httppkg.MakeHTTPHandlerFunc(controller.APIProxyTraffic)).Methods(http.MethodGet)
	return router
}

func performRequest(handler http.Handler, target string) *httptest.ResponseRecorder {
	return performRequestWithMethod(handler, http.MethodGet, target)
}

func performRequestWithMethod(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
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

func assertRawJSONKeys(t *testing.T, raw map[string]json.RawMessage, want ...string) {
	t.Helper()

	if len(raw) != len(want) {
		t.Fatalf("json keys mismatch, want %v got %v", want, raw)
	}
	for _, key := range want {
		if _, ok := raw[key]; !ok {
			t.Fatalf("json key %q missing from %v", key, raw)
		}
	}
}

func assertRawJSONKeysFromMessage(t *testing.T, raw json.RawMessage, want ...string) {
	t.Helper()

	var out map[string]json.RawMessage
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal raw json object failed: %v, body: %s", err, string(raw))
	}
	assertRawJSONKeys(t, out, want...)
}

func mustMarshalJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	out, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}
	return out
}
