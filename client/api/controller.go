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
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/fatedier/frp/client/proxy"
	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/policy/security"
	httppkg "github.com/fatedier/frp/pkg/util/http"
	"github.com/fatedier/frp/pkg/util/log"
)

// Controller handles HTTP API requests for frpc.
type Controller struct {
	// getProxyStatus returns the current proxy status.
	// Returns nil if the control connection is not established.
	getProxyStatus func() []*proxy.WorkingStatus

	// serverAddr is the frps server address for display.
	serverAddr string

	// configFilePath is the path to the configuration file.
	configFilePath string

	// unsafeFeatures is used for validation.
	unsafeFeatures *security.UnsafeFeatures

	// updateConfig updates proxy and visitor configurations.
	updateConfig func(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error

	// gracefulClose gracefully stops the service.
	gracefulClose func(d time.Duration)
}

// ControllerParams contains parameters for creating an APIController.
type ControllerParams struct {
	GetProxyStatus func() []*proxy.WorkingStatus
	ServerAddr     string
	ConfigFilePath string
	UnsafeFeatures *security.UnsafeFeatures
	UpdateConfig   func(proxyCfgs []v1.ProxyConfigurer, visitorCfgs []v1.VisitorConfigurer) error
	GracefulClose  func(d time.Duration)
}

// NewController creates a new Controller.
func NewController(params ControllerParams) *Controller {
	return &Controller{
		getProxyStatus: params.GetProxyStatus,
		serverAddr:     params.ServerAddr,
		configFilePath: params.ConfigFilePath,
		unsafeFeatures: params.UnsafeFeatures,
		updateConfig:   params.UpdateConfig,
		gracefulClose:  params.GracefulClose,
	}
}

// Reload handles GET /api/reload
func (c *Controller) Reload(ctx *httppkg.Context) (any, error) {
	strictConfigMode := false
	strictStr := ctx.Query("strictConfig")
	if strictStr != "" {
		strictConfigMode, _ = strconv.ParseBool(strictStr)
	}

	cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(c.configFilePath, strictConfigMode)
	if err != nil {
		log.Warnf("reload frpc proxy config error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	if _, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs, c.unsafeFeatures); err != nil {
		log.Warnf("reload frpc proxy config error: %s", err.Error())
		return nil, httppkg.NewError(http.StatusBadRequest, err.Error())
	}

	if err := c.updateConfig(proxyCfgs, visitorCfgs); err != nil {
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

// buildProxyStatusResp creates a ProxyStatusResp from proxy.WorkingStatus
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
	return psr
}
