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

package api

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/client/proxy"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

// Controller handles HTTP API requests for frpc.
type Controller struct {
	serverAddr string
	manager    configmgmt.ConfigManager
}

// ControllerParams contains parameters for creating an APIController.
type ControllerParams struct {
	ServerAddr string
	Manager    configmgmt.ConfigManager
}

func NewController(params ControllerParams) *Controller {
	return &Controller{
		serverAddr: params.ServerAddr,
		manager:    params.Manager,
	}
}

func (c *Controller) toHTTPError(err error) error {
	if err == nil {
		return nil
	}

	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, configmgmt.ErrInvalidArgument):
		code = http.StatusBadRequest
	case errors.Is(err, configmgmt.ErrNotFound), errors.Is(err, configmgmt.ErrStoreDisabled):
		code = http.StatusNotFound
	case errors.Is(err, configmgmt.ErrConflict):
		code = http.StatusConflict
	}
	return httppkg.NewError(code, err.Error())
}

// TODO(fatedier): Remove this lock wrapper after migrating typed config
// decoding to encoding/json/v2 with per-call options.
// TypedProxyConfig/TypedVisitorConfig currently read global strictness state.
func unmarshalTypedConfig[T any](body []byte, out *T) error {
	return v1.WithDisallowUnknownFields(false, func() error {
		return json.Unmarshal(body, out)
	})
}

// Reload handles GET /api/reload
func (c *Controller) Reload(ctx *httppkg.Context) (any, error) {
	strictConfigMode := false
	strictStr := ctx.Query("strictConfig")
	if strictStr != "" {
		strictConfigMode, _ = strconv.ParseBool(strictStr)
	}

	if err := c.manager.ReloadFromFile(strictConfigMode); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

// Stop handles POST /api/stop
func (c *Controller) Stop(ctx *httppkg.Context) (any, error) {
	go c.manager.GracefulClose(100 * time.Millisecond)
	return nil, nil
}

// Status handles GET /api/status
func (c *Controller) Status(ctx *httppkg.Context) (any, error) {
	res := make(StatusResp)
	ps := c.manager.GetProxyStatus()
	if ps == nil {
		return res, nil
	}

	for _, status := range ps {
		res[status.Type] = append(res[status.Type], c.buildProxyStatusResp(status))
	}

	for _, arrs := range res {
		if len(arrs) <= 1 {
			continue
		}
		slices.SortFunc(arrs, func(a, b ProxyStatusResp) int {
			return cmp.Compare(a.Name, b.Name)
		})
	}
	return res, nil
}

// GetConfig handles GET /api/config
func (c *Controller) GetConfig(ctx *httppkg.Context) (any, error) {
	content, err := c.manager.ReadConfigFile()
	if err != nil {
		return nil, c.toHTTPError(err)
	}
	return content, nil
}

// PutConfig handles PUT /api/config
func (c *Controller) PutConfig(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read request body error: %v", err))
	}

	if len(body) == 0 {
		return nil, httppkg.NewError(http.StatusBadRequest, "body can't be empty")
	}

	if err := c.manager.WriteConfigFile(body); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) buildProxyStatusResp(status *proxy.WorkingStatus) ProxyStatusResp {
	psr := ProxyStatusResp{
		Name:   status.Name,
		Type:   status.Type,
		Status: status.Phase,
		Err:    status.Err,
	}
	baseCfg := status.Cfg.GetBaseConfig()
	if baseCfg.LocalPort != 0 {
		psr.LocalAddr = net.JoinHostPort(baseCfg.LocalIP, strconv.Itoa(baseCfg.LocalPort))
	}
	psr.Plugin = baseCfg.Plugin.Type

	if status.Err == "" {
		psr.RemoteAddr = status.RemoteAddr
		if slices.Contains([]string{"tcp", "udp"}, status.Type) {
			psr.RemoteAddr = c.serverAddr + psr.RemoteAddr
		}
	}

	if c.manager.IsStoreProxyEnabled(status.Name) {
		psr.Source = SourceStore
	}
	return psr
}

func (c *Controller) ListStoreProxies(ctx *httppkg.Context) (any, error) {
	proxies, err := c.manager.ListStoreProxies()
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp := ProxyListResp{Proxies: make([]ProxyConfig, 0, len(proxies))}
	for _, p := range proxies {
		cfg, err := configurerToMap(p)
		if err != nil {
			continue
		}
		resp.Proxies = append(resp.Proxies, ProxyConfig{
			Name:   p.GetBaseConfig().Name,
			Type:   p.GetBaseConfig().Type,
			Config: cfg,
		})
	}
	return resp, nil
}

func (c *Controller) GetStoreProxy(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name is required")
	}

	p, err := c.manager.GetStoreProxy(name)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	cfg, err := configurerToMap(p)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	return ProxyConfig{
		Name:   p.GetBaseConfig().Name,
		Type:   p.GetBaseConfig().Type,
		Config: cfg,
	}, nil
}

func (c *Controller) CreateStoreProxy(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var typed v1.TypedProxyConfig
	if err := unmarshalTypedConfig(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.ProxyConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid proxy config: type is required")
	}

	if err := c.manager.CreateStoreProxy(typed.ProxyConfigurer); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) UpdateStoreProxy(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name is required")
	}

	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var typed v1.TypedProxyConfig
	if err := unmarshalTypedConfig(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.ProxyConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid proxy config: type is required")
	}

	if err := c.manager.UpdateStoreProxy(name, typed.ProxyConfigurer); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) DeleteStoreProxy(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name is required")
	}

	if err := c.manager.DeleteStoreProxy(name); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) ListStoreVisitors(ctx *httppkg.Context) (any, error) {
	visitors, err := c.manager.ListStoreVisitors()
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp := VisitorListResp{Visitors: make([]VisitorConfig, 0, len(visitors))}
	for _, v := range visitors {
		cfg, err := configurerToMap(v)
		if err != nil {
			continue
		}
		resp.Visitors = append(resp.Visitors, VisitorConfig{
			Name:   v.GetBaseConfig().Name,
			Type:   v.GetBaseConfig().Type,
			Config: cfg,
		})
	}
	return resp, nil
}

func (c *Controller) GetStoreVisitor(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name is required")
	}

	v, err := c.manager.GetStoreVisitor(name)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	cfg, err := configurerToMap(v)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	return VisitorConfig{
		Name:   v.GetBaseConfig().Name,
		Type:   v.GetBaseConfig().Type,
		Config: cfg,
	}, nil
}

func (c *Controller) CreateStoreVisitor(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var typed v1.TypedVisitorConfig
	if err := unmarshalTypedConfig(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.VisitorConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid visitor config: type is required")
	}

	if err := c.manager.CreateStoreVisitor(typed.VisitorConfigurer); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) UpdateStoreVisitor(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name is required")
	}

	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var typed v1.TypedVisitorConfig
	if err := unmarshalTypedConfig(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.VisitorConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid visitor config: type is required")
	}

	if err := c.manager.UpdateStoreVisitor(name, typed.VisitorConfigurer); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func (c *Controller) DeleteStoreVisitor(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name is required")
	}

	if err := c.manager.DeleteStoreVisitor(name); err != nil {
		return nil, c.toHTTPError(err)
	}
	return nil, nil
}

func configurerToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
