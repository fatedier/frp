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
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/source"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
)

// Controller handles HTTP API requests for frpc.
type Controller struct {
	getProxyStatus    func() []*proxy.WorkingStatus
	serverAddr        string
	configFilePath    string
	unsafeFeatures    *security.UnsafeFeatures
	updateConfig      func(common *v1.ClientCommonConfig, proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error
	reloadFromSources func() error
	gracefulClose     func(d time.Duration)
	storeSource       *source.StoreSource
}

// ControllerParams contains parameters for creating an APIController.
type ControllerParams struct {
	GetProxyStatus    func() []*proxy.WorkingStatus
	ServerAddr        string
	ConfigFilePath    string
	UnsafeFeatures    *security.UnsafeFeatures
	UpdateConfig      func(common *v1.ClientCommonConfig, proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error
	ReloadFromSources func() error
	GracefulClose     func(d time.Duration)
	StoreSource       *source.StoreSource
}

func NewController(params ControllerParams) *Controller {
	return &Controller{
		getProxyStatus:    params.GetProxyStatus,
		serverAddr:        params.ServerAddr,
		configFilePath:    params.ConfigFilePath,
		unsafeFeatures:    params.UnsafeFeatures,
		updateConfig:      params.UpdateConfig,
		reloadFromSources: params.ReloadFromSources,
		gracefulClose:     params.GracefulClose,
		storeSource:       params.StoreSource,
	}
}

func (c *Controller) reloadFromSourcesOrError() error {
	if err := c.reloadFromSources(); err != nil {
		return httppkg.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to apply config: %v", err))
	}
	return nil
}

// Reload handles GET /api/reload
func (c *Controller) Reload(ctx *httppkg.Context) (any, error) {
	strictConfigMode := false
	strictStr := ctx.Query("strictConfig")
	if strictStr != "" {
		strictConfigMode, _ = strconv.ParseBool(strictStr)
	}

	result, err := config.LoadClientConfigResult(c.configFilePath, strictConfigMode)
	if err != nil {
		log.Warnf("reload frpc proxy config error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	proxyCfgs := result.Proxies
	visitorCfgs := result.Visitors

	proxyCfgsForValidation, visitorCfgsForValidation := config.FilterClientConfigurers(
		result.Common,
		proxyCfgs,
		visitorCfgs,
	)
	proxyCfgsForValidation = config.CompleteProxyConfigurers(proxyCfgsForValidation)
	visitorCfgsForValidation = config.CompleteVisitorConfigurers(visitorCfgsForValidation)

	if _, err := validation.ValidateAllClientConfig(result.Common, proxyCfgsForValidation, visitorCfgsForValidation, c.unsafeFeatures); err != nil {
		log.Warnf("reload frpc proxy config error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	if err := c.updateConfig(result.Common, proxyCfgs, visitorCfgs); err != nil {
		log.Warnf("reload frpc proxy config error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusInternalServerError, err.Error())
	}

	log.Infof("success reload conf")
	return nil, nil
}

// Stop handles POST /api/stop
func (c *Controller) Stop(ctx *httppkg.Context) (any, error) {
	go c.gracefulClose(100 * time.Millisecond)
	return nil, nil
}

// Status handles GET /api/status
func (c *Controller) Status(ctx *httppkg.Context) (any, error) {
	res := make(StatusResp)
	ps := c.getProxyStatus()
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
	if c.configFilePath == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "frpc has no config file path")
	}

	content, err := os.ReadFile(c.configFilePath)
	if err != nil {
		log.Warnf("load frpc config file error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}
	return string(content), nil
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

	if err := os.WriteFile(c.configFilePath, body, 0o600); err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, fmt.Sprintf("write content to frpc config file error: %v", err))
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

	// Check if proxy is from store
	if c.storeSource != nil {
		if c.storeSource.GetProxy(status.Name) != nil {
			psr.Source = "store"
		}
	}
	return psr
}

func (c *Controller) ListStoreProxies(ctx *httppkg.Context) (any, error) {
	proxies, err := c.storeSource.GetAllProxies()
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to list proxies: %v", err))
	}
	resp := ProxyListResp{Proxies: make([]ProxyConfig, 0, len(proxies))}

	for _, p := range proxies {
		cfg, err := proxyConfigurerToMap(p)
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

	p := c.storeSource.GetProxy(name)
	if p == nil {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("proxy %q not found", name))
	}

	cfg, err := proxyConfigurerToMap(p)
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
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.ProxyConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid proxy config: type is required")
	}

	typed.Complete()
	if err := validation.ValidateProxyConfigurerForClient(typed.ProxyConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
	}

	if err := c.storeSource.AddProxy(typed.ProxyConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusConflict, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: created proxy %q", typed.ProxyConfigurer.GetBaseConfig().Name)
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
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.ProxyConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid proxy config: type is required")
	}

	bodyName := typed.ProxyConfigurer.GetBaseConfig().Name
	if bodyName != name {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name in URL must match name in body")
	}

	typed.Complete()
	if err := validation.ValidateProxyConfigurerForClient(typed.ProxyConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
	}

	if err := c.storeSource.UpdateProxy(typed.ProxyConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusNotFound, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: updated proxy %q", name)
	return nil, nil
}

func (c *Controller) DeleteStoreProxy(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "proxy name is required")
	}

	if err := c.storeSource.RemoveProxy(name); err != nil {
		return nil, httppkg.NewError(http.StatusNotFound, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: deleted proxy %q", name)
	return nil, nil
}

func (c *Controller) ListStoreVisitors(ctx *httppkg.Context) (any, error) {
	visitors, err := c.storeSource.GetAllVisitors()
	if err != nil {
		return nil, httppkg.NewError(http.StatusInternalServerError, fmt.Sprintf("failed to list visitors: %v", err))
	}
	resp := VisitorListResp{Visitors: make([]VisitorConfig, 0, len(visitors))}

	for _, v := range visitors {
		cfg, err := visitorConfigurerToMap(v)
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

	v := c.storeSource.GetVisitor(name)
	if v == nil {
		return nil, httppkg.NewError(http.StatusNotFound, fmt.Sprintf("visitor %q not found", name))
	}

	cfg, err := visitorConfigurerToMap(v)
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
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.VisitorConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid visitor config: type is required")
	}

	typed.Complete()
	if err := validation.ValidateVisitorConfigurer(typed.VisitorConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
	}

	if err := c.storeSource.AddVisitor(typed.VisitorConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusConflict, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: created visitor %q", typed.VisitorConfigurer.GetBaseConfig().Name)
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
	if err := json.Unmarshal(body, &typed); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("parse JSON error: %v", err))
	}

	if typed.VisitorConfigurer == nil {
		return nil, httppkg.NewError(http.StatusBadRequest, "invalid visitor config: type is required")
	}

	bodyName := typed.VisitorConfigurer.GetBaseConfig().Name
	if bodyName != name {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name in URL must match name in body")
	}

	typed.Complete()
	if err := validation.ValidateVisitorConfigurer(typed.VisitorConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
	}

	if err := c.storeSource.UpdateVisitor(typed.VisitorConfigurer); err != nil {
		return nil, httppkg.NewError(http.StatusNotFound, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: updated visitor %q", name)
	return nil, nil
}

func (c *Controller) DeleteStoreVisitor(ctx *httppkg.Context) (any, error) {
	name := ctx.Param("name")
	if name == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "visitor name is required")
	}

	if err := c.storeSource.RemoveVisitor(name); err != nil {
		return nil, httppkg.NewError(http.StatusNotFound, err.Error())
	}
	if err := c.reloadFromSourcesOrError(); err != nil {
		return nil, err
	}

	log.Infof("store: deleted visitor %q", name)
	return nil, nil
}

func proxyConfigurerToMap(p v1.ProxyConfigurer) (map[string]any, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func visitorConfigurerToMap(v v1.VisitorConfigurer) (map[string]any, error) {
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
