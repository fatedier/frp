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

package http

import (
	"cmp"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/fatedier/frp/client/configmgmt"
	"github.com/fatedier/frp/client/http/model"
	"github.com/fatedier/frp/client/proxy"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/jsonx"
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
	res := make(model.StatusResp)
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
		slices.SortFunc(arrs, func(a, b model.ProxyStatusResp) int {
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

func (c *Controller) buildProxyStatusResp(status *proxy.WorkingStatus) model.ProxyStatusResp {
	psr := model.ProxyStatusResp{
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
		psr.Source = model.SourceStore
	}
	return psr
}

// GetProxyConfig handles GET /api/proxy/{name}/config
func (c *Controller) GetProxyConfig(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name is required")
	}

	cfg, ok := c.manager.GetProxyConfig(name)
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("proxy %q not found", name))
	}

	payload, err := model.ProxyDefinitionFromConfigurer(cfg)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return payload, nil
}

// GetVisitorConfig handles GET /api/visitor/{name}/config
func (c *Controller) GetVisitorConfig(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name is required")
	}

	cfg, ok := c.manager.GetVisitorConfig(name)
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("visitor %q not found", name))
	}

	payload, err := model.VisitorDefinitionFromConfigurer(cfg)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return payload, nil
}

func (c *Controller) ListStoreProxies(ctx *httppkg.Context) (any, error) {
	proxies, err := c.manager.ListStoreProxies()
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp := model.ProxyListResp{Proxies: make([]model.ProxyDefinition, 0, len(proxies))}
	for _, p := range proxies {
		payload, err := model.ProxyDefinitionFromConfigurer(p)
		if err != nil {
			return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
		}
		resp.Proxies = append(resp.Proxies, payload)
	}
	slices.SortFunc(resp.Proxies, func(a, b model.ProxyDefinition) int {
		return cmp.Compare(a.Name, b.Name)
	})
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

	payload, err := model.ProxyDefinitionFromConfigurer(p)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	return payload, nil
}

func (c *Controller) CreateStoreProxy(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var payload model.ProxyDefinition
	if err := jsonx.Unmarshal(body, &payload); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if err := payload.Validate("", false); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	cfg, err := payload.ToConfigurer()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	created, err := c.manager.CreateStoreProxy(cfg)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp, err := model.ProxyDefinitionFromConfigurer(created)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return resp, nil
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

	var payload model.ProxyDefinition
	if err := jsonx.Unmarshal(body, &payload); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if err := payload.Validate(name, true); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	cfg, err := payload.ToConfigurer()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	updated, err := c.manager.UpdateStoreProxy(name, cfg)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp, err := model.ProxyDefinitionFromConfigurer(updated)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return resp, nil
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

	resp := model.VisitorListResp{Visitors: make([]model.VisitorDefinition, 0, len(visitors))}
	for _, v := range visitors {
		payload, err := model.VisitorDefinitionFromConfigurer(v)
		if err != nil {
			return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
		}
		resp.Visitors = append(resp.Visitors, payload)
	}
	slices.SortFunc(resp.Visitors, func(a, b model.VisitorDefinition) int {
		return cmp.Compare(a.Name, b.Name)
	})
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

	payload, err := model.VisitorDefinitionFromConfigurer(v)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	return payload, nil
}

func (c *Controller) CreateStoreVisitor(ctx *httppkg.Context) (any, error) {
	body, err := ctx.Body()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read body error: %v", err))
	}

	var payload model.VisitorDefinition
	if err := jsonx.Unmarshal(body, &payload); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if err := payload.Validate("", false); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	cfg, err := payload.ToConfigurer()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	created, err := c.manager.CreateStoreVisitor(cfg)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp, err := model.VisitorDefinitionFromConfigurer(created)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return resp, nil
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

	var payload model.VisitorDefinition
	if err := jsonx.Unmarshal(body, &payload); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if err := payload.Validate(name, true); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	cfg, err := payload.ToConfigurer()
	if err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	updated, err := c.manager.UpdateStoreVisitor(name, cfg)
	if err != nil {
		return nil, c.toHTTPError(err)
	}

	resp, err := model.VisitorDefinitionFromConfigurer(updated)
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}
	return resp, nil
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
