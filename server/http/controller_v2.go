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
	"cmp"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/metrics/mem"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/server/http/model"
	"github.com/fatedier/frp/server/registry"
)

const (
	defaultV2Page     = 1
	defaultV2PageSize = 50
	maxV2PageSize     = 200

	v2SystemPruneTypeOfflineProxies = "offline_proxies"
	v2ProxyTrafficDefaultDays       = 7
	v2ProxyTrafficUnit              = "bytes"
	v2ProxyTrafficGranularity       = "day"
)

var apiV2ProxyTypes = []string{
	string(v1.ProxyTypeTCP),
	string(v1.ProxyTypeUDP),
	string(v1.ProxyTypeHTTP),
	string(v1.ProxyTypeHTTPS),
	string(v1.ProxyTypeTCPMUX),
	string(v1.ProxyTypeSTCP),
	string(v1.ProxyTypeXTCP),
	string(v1.ProxyTypeSUDP),
}

// /api/v2/system/info
func (c *Controller) APIV2SystemInfo(ctx *httppkg.Context) (any, error) {
	info := c.buildServerInfoResp()
	proxyTypeCounts := info.ProxyTypeCounts
	if proxyTypeCounts == nil {
		proxyTypeCounts = map[string]int64{}
	}

	return model.V2SystemInfoResp{
		Version: info.Version,
		Config: model.V2SystemInfoConfigResp{
			BindPort:              info.BindPort,
			VhostHTTPPort:         info.VhostHTTPPort,
			VhostHTTPSPort:        info.VhostHTTPSPort,
			TCPMuxHTTPConnectPort: info.TCPMuxHTTPConnectPort,
			KCPBindPort:           info.KCPBindPort,
			QUICBindPort:          info.QUICBindPort,
			SubdomainHost:         info.SubdomainHost,
			MaxPoolCount:          info.MaxPoolCount,
			MaxPortsPerClient:     info.MaxPortsPerClient,
			HeartbeatTimeout:      info.HeartBeatTimeout,
			AllowPortsStr:         info.AllowPortsStr,
			TLSForce:              info.TLSForce,
		},
		Status: model.V2SystemInfoStatusResp{
			TotalTrafficIn:  info.TotalTrafficIn,
			TotalTrafficOut: info.TotalTrafficOut,
			CurConns:        info.CurConns,
			ClientCounts:    info.ClientCounts,
			ProxyTypeCounts: proxyTypeCounts,
		},
	}, nil
}

// /api/v2/system/prune
func (c *Controller) APIV2SystemPrune(ctx *httppkg.Context) (any, error) {
	pruneType, err := parseV2SystemPruneType(ctx.Query("type"))
	if err != nil {
		return nil, err
	}

	cleared, total := mem.StatsCollector.PruneOfflineProxies()
	return model.V2SystemPruneResp{
		Type:    pruneType,
		Cleared: cleared,
		Total:   total,
	}, nil
}

// /api/v2/users
func (c *Controller) APIV2UserList(ctx *httppkg.Context) (any, error) {
	page, pageSize, err := parseV2PageParams(ctx)
	if err != nil {
		return nil, err
	}
	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}

	userStats := make(map[string]*model.V2UserResp)
	for _, info := range c.clientRegistry.List() {
		item := getOrCreateV2User(userStats, info.User)
		item.ClientCount++
	}
	for _, proxyInfo := range c.listV2ProxyStats("") {
		item := getOrCreateV2User(userStats, proxyInfo.User)
		item.ProxyCount++
	}

	q := strings.ToLower(ctx.Query("q"))
	items := make([]model.V2UserResp, 0, len(userStats))
	for _, item := range userStats {
		if q != "" && !strings.Contains(strings.ToLower(item.User), q) {
			continue
		}
		items = append(items, *item)
	}
	slices.SortFunc(items, func(a, b model.V2UserResp) int {
		return cmp.Compare(a.User, b.User)
	})

	return buildV2PageResp(items, page, pageSize), nil
}

// /api/v2/clients
func (c *Controller) APIV2ClientList(ctx *httppkg.Context) (any, error) {
	page, pageSize, err := parseV2PageParams(ctx)
	if err != nil {
		return nil, err
	}
	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}
	statusFilter, err := parseV2StatusFilter(ctx.Query("status"))
	if err != nil {
		return nil, err
	}

	userFilter, filterByUser := queryValue(ctx, "user")
	clientIDFilter := ctx.Query("clientID")
	runIDFilter := ctx.Query("runID")
	q := strings.ToLower(ctx.Query("q"))

	records := c.clientRegistry.List()
	items := make([]model.ClientInfoResp, 0, len(records))
	for _, info := range records {
		if filterByUser && info.User != userFilter {
			continue
		}
		if clientIDFilter != "" && info.ClientID() != clientIDFilter {
			continue
		}
		if runIDFilter != "" && info.RunID != runIDFilter {
			continue
		}
		if !matchV2StatusFilter(info.Online, statusFilter) {
			continue
		}
		resp := buildClientInfoResp(info)
		if q != "" && !matchV2ClientQuery(resp, q) {
			continue
		}
		items = append(items, resp)
	}

	slices.SortFunc(items, func(a, b model.ClientInfoResp) int {
		if v := cmp.Compare(a.User, b.User); v != 0 {
			return v
		}
		if v := cmp.Compare(a.ClientID, b.ClientID); v != 0 {
			return v
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return buildV2PageResp(items, page, pageSize), nil
}

// /api/v2/clients/{key}
func (c *Controller) APIV2ClientDetail(ctx *httppkg.Context) (any, error) {
	key, err := decodeV2PathParam(ctx, "key", "client key")
	if err != nil {
		return nil, err
	}

	if c.clientRegistry == nil {
		return nil, fmt.Errorf("client registry unavailable")
	}

	info, ok := c.clientRegistry.GetByKey(key)
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("client %s not found", key))
	}

	resp := buildClientInfoResp(info)
	status := c.buildV2ClientStatus(info)
	return model.V2ClientDetailResp{
		ClientInfoResp: resp,
		Status:         status,
	}, nil
}

// /api/v2/proxies
func (c *Controller) APIV2ProxyList(ctx *httppkg.Context) (any, error) {
	page, pageSize, err := parseV2PageParams(ctx)
	if err != nil {
		return nil, err
	}
	statusFilter, err := parseV2StatusFilter(ctx.Query("status"))
	if err != nil {
		return nil, err
	}

	proxyType, err := parseV2ProxyTypeFilter(ctx.Query("type"))
	if err != nil {
		return nil, err
	}
	userFilter, filterByUser := queryValue(ctx, "user")
	clientIDFilter := ctx.Query("clientID")
	q := strings.ToLower(ctx.Query("q"))

	stats := c.listV2ProxyStats(proxyType)
	items := make([]model.V2ProxyResp, 0, len(stats))
	for _, ps := range stats {
		resp := c.buildV2ProxyResp(ps)
		if filterByUser && resp.User != userFilter {
			continue
		}
		if clientIDFilter != "" && resp.ClientID != clientIDFilter {
			continue
		}
		if !matchV2StatusFilter(resp.Status.State == "online", statusFilter) {
			continue
		}
		if q != "" && !matchV2ProxyQuery(resp, q) {
			continue
		}
		items = append(items, resp)
	}

	slices.SortFunc(items, func(a, b model.V2ProxyResp) int {
		if v := cmp.Compare(a.Type, b.Type); v != 0 {
			return v
		}
		return cmp.Compare(a.Name, b.Name)
	})

	return buildV2PageResp(items, page, pageSize), nil
}

// /api/v2/proxies/{name}
func (c *Controller) APIV2ProxyDetail(ctx *httppkg.Context) (any, error) {
	name, err := decodeV2PathParam(ctx, "name", "proxy name")
	if err != nil {
		return nil, err
	}

	ps := mem.StatsCollector.GetProxyByName(name)
	if ps == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "no proxy info found")
	}
	return c.buildV2ProxyResp(ps), nil
}

// /api/v2/proxies/{name}/traffic
func (c *Controller) APIV2ProxyTraffic(ctx *httppkg.Context) (any, error) {
	name, err := decodeV2PathParam(ctx, "name", "proxy name")
	if err != nil {
		return nil, err
	}

	proxyTrafficInfo := mem.StatsCollector.GetProxyTraffic(name)
	if proxyTrafficInfo == nil {
		return nil, httppkg.NewError(http.StatusNotFound, "no proxy info found")
	}

	return buildV2ProxyTrafficResp(name, proxyTrafficInfo, time.Now()), nil
}

func decodeV2PathParam(ctx *httppkg.Context, key string, label string) (string, error) {
	raw := ctx.Param(key)
	if raw == "" {
		return "", fmt.Errorf("missing %s", label)
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("invalid %s", label))
	}
	return decoded, nil
}

func getOrCreateV2User(items map[string]*model.V2UserResp, user string) *model.V2UserResp {
	item, ok := items[user]
	if !ok {
		item = &model.V2UserResp{User: user}
		items[user] = item
	}
	return item
}

func parseV2PageParams(ctx *httppkg.Context) (int, int, error) {
	page, err := parseV2PositiveInt(ctx.Query("page"), defaultV2Page, "page")
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := parseV2PositiveInt(ctx.Query("pageSize"), defaultV2PageSize, "pageSize")
	if err != nil {
		return 0, 0, err
	}
	if pageSize > maxV2PageSize {
		return 0, 0, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("pageSize must be between 1 and %d", maxV2PageSize))
	}
	if page > math.MaxInt/pageSize {
		return 0, 0, httppkg.NewError(http.StatusBadRequest, "page is too large")
	}
	return page, pageSize, nil
}

func parseV2PositiveInt(raw string, defaultValue int, name string) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("%s must be a positive integer", name))
	}
	return value, nil
}

func parseV2StatusFilter(raw string) (string, error) {
	status := strings.ToLower(raw)
	switch status {
	case "", "all", "online", "offline":
		return status, nil
	default:
		return "", httppkg.NewError(http.StatusBadRequest, "status must be one of all, online, offline")
	}
}

func parseV2ProxyTypeFilter(raw string) (string, error) {
	proxyType := strings.ToLower(raw)
	if proxyType == "" {
		return "", nil
	}
	if slices.Contains(apiV2ProxyTypes, proxyType) {
		return proxyType, nil
	}
	return "", httppkg.NewError(http.StatusBadRequest, "type must be one of tcp, udp, http, https, tcpmux, stcp, xtcp, sudp")
}

func parseV2SystemPruneType(raw string) (string, error) {
	pruneType := strings.ToLower(raw)
	switch pruneType {
	case "":
		return "", httppkg.NewError(http.StatusBadRequest, "type is required")
	case v2SystemPruneTypeOfflineProxies:
		return pruneType, nil
	default:
		return "", httppkg.NewError(http.StatusBadRequest, "type must be one of offline_proxies")
	}
}

func matchV2StatusFilter(online bool, filter string) bool {
	switch filter {
	case "", "all":
		return true
	case "online":
		return online
	case "offline":
		return !online
	default:
		return true
	}
}

func buildV2PageResp[T any](items []T, page, pageSize int) model.V2PageResp[T] {
	total := len(items)
	return model.V2PageResp[T]{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    paginateV2Items(items, page, pageSize),
	}
}

func paginateV2Items[T any](items []T, page, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := min(start+pageSize, len(items))
	return items[start:end]
}

func queryValue(ctx *httppkg.Context, key string) (string, bool) {
	values, ok := ctx.Req.URL.Query()[key]
	if !ok {
		return "", false
	}
	if len(values) == 0 {
		return "", true
	}
	return values[0], true
}

func matchV2ClientQuery(item model.ClientInfoResp, q string) bool {
	return containsV2Query(q,
		item.Key,
		item.User,
		item.ClientID,
		item.RunID,
		item.Version,
		item.WireProtocol,
		item.Hostname,
		item.ClientIP,
	)
}

func matchV2ProxyQuery(item model.V2ProxyResp, q string) bool {
	values := []string{
		item.Name,
		item.Type,
		item.User,
		item.ClientID,
		item.Status.State,
	}

	switch spec := item.Spec.(type) {
	case *model.TCPOutConf:
		values = append(values, strconv.Itoa(spec.RemotePort))
	case *model.UDPOutConf:
		values = append(values, strconv.Itoa(spec.RemotePort))
	case *model.HTTPOutConf:
		values = append(values, spec.CustomDomains...)
		values = append(values, spec.SubDomain)
	case *model.HTTPSOutConf:
		values = append(values, spec.CustomDomains...)
		values = append(values, spec.SubDomain)
	case *model.TCPMuxOutConf:
		values = append(values, spec.CustomDomains...)
		values = append(values, spec.SubDomain)
	}

	return containsV2Query(q, values...)
}

func containsV2Query(q string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), q) {
			return true
		}
	}
	return false
}

func (c *Controller) listV2ProxyStats(proxyType string) []*mem.ProxyStats {
	if proxyType != "" {
		return mem.StatsCollector.GetProxiesByType(proxyType)
	}

	items := make([]*mem.ProxyStats, 0)
	for _, t := range apiV2ProxyTypes {
		items = append(items, mem.StatsCollector.GetProxiesByType(t)...)
	}
	return items
}

func buildV2ProxyTrafficResp(name string, traffic *mem.ProxyTrafficInfo, now time.Time) model.V2ProxyTrafficResp {
	history := make([]model.V2ProxyTrafficPointResp, 0, v2ProxyTrafficDefaultDays)
	for age := v2ProxyTrafficDefaultDays - 1; age >= 0; age-- {
		history = append(history, model.V2ProxyTrafficPointResp{
			Date:       now.AddDate(0, 0, -age).Format(time.DateOnly),
			TrafficIn:  v2TrafficValueAt(traffic.TrafficIn, age),
			TrafficOut: v2TrafficValueAt(traffic.TrafficOut, age),
		})
	}

	return model.V2ProxyTrafficResp{
		Name:        name,
		Unit:        v2ProxyTrafficUnit,
		Granularity: v2ProxyTrafficGranularity,
		History:     history,
	}
}

func v2TrafficValueAt(values []int64, todayFirstIndex int) int64 {
	if todayFirstIndex >= len(values) {
		return 0
	}
	return values[todayFirstIndex]
}

func (c *Controller) buildV2ClientStatus(info registry.ClientInfo) model.V2ClientStatusResp {
	status := model.V2ClientStatusResp{State: "offline"}
	if info.Online {
		status.State = "online"
	}

	user := info.User
	clientID := info.ClientID()
	for _, ps := range c.listV2ProxyStats("") {
		if ps.User != user || ps.ClientID != clientID {
			continue
		}
		status.CurConns += ps.CurConns
		status.ProxyCount++
	}
	return status
}

func (c *Controller) buildV2ProxyResp(ps *mem.ProxyStats) model.V2ProxyResp {
	state := "offline"
	var spec any
	if c.pxyManager != nil {
		if pxy, ok := c.pxyManager.GetByName(ps.Name); ok {
			state = "online"
			spec = getConfFromConfigurer(pxy.GetConfigurer())
		}
	}

	return model.V2ProxyResp{
		Name:     ps.Name,
		Type:     ps.Type,
		User:     ps.User,
		ClientID: ps.ClientID,
		Spec:     spec,
		Status: model.V2ProxyStatusResp{
			State:           state,
			TodayTrafficIn:  ps.TodayTrafficIn,
			TodayTrafficOut: ps.TodayTrafficOut,
			CurConns:        ps.CurConns,
			LastStartAt:     ps.LastStartAt,
			LastCloseAt:     ps.LastCloseAt,
		},
	}
}
