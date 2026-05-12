// Copyright 2019 fatedier, fatedier@gmail.com
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

package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/xlog"
)

type Manager struct {
	loginPlugins       []Plugin
	newProxyPlugins    []Plugin
	closeProxyPlugins  []Plugin
	pingPlugins        []Plugin
	newWorkConnPlugins []Plugin
	newUserConnPlugins []Plugin
}

func NewManager() *Manager {
	return &Manager{
		loginPlugins:       make([]Plugin, 0),
		newProxyPlugins:    make([]Plugin, 0),
		closeProxyPlugins:  make([]Plugin, 0),
		pingPlugins:        make([]Plugin, 0),
		newWorkConnPlugins: make([]Plugin, 0),
		newUserConnPlugins: make([]Plugin, 0),
	}
}

func newPluginRequestContext() (context.Context, *xlog.Logger) {
	reqid, _ := util.RandID()
	xl := xlog.New().AppendPrefix("reqid: " + reqid)
	ctx := xlog.NewContext(context.Background(), xl)
	return NewReqidContext(ctx, reqid), xl
}

type pluginErrorLogMode bool

const (
	// Warn is the zero value because it is the default for mutable plugin operations.
	pluginErrorLogWarn pluginErrorLogMode = false
	pluginErrorLogInfo pluginErrorLogMode = true
)

func logPluginError(xl *xlog.Logger, p Plugin, op string, err error, mode pluginErrorLogMode) {
	if mode == pluginErrorLogInfo {
		xl.Infof("send %s request to plugin [%s] error: %v", op, p.Name(), err)
		return
	}
	xl.Warnf("send %s request to plugin [%s] error: %v", op, p.Name(), err)
}

func handleMutableContent[T any](
	plugins []Plugin,
	op string,
	content *T,
	logMode pluginErrorLogMode,
) (*T, error) {
	if len(plugins) == 0 {
		return content, nil
	}

	var (
		res = &Response{
			Reject:   false,
			Unchange: true,
		}
		retContent any
		err        error
	)
	ctx, xl := newPluginRequestContext()

	for _, p := range plugins {
		res, retContent, err = p.Handle(ctx, op, *content)
		if err != nil {
			logPluginError(xl, p, op, err, logMode)
			return nil, errors.New("send " + op + " request to plugin error")
		}
		if res.Reject {
			return nil, fmt.Errorf("%s", res.RejectReason)
		}
		if !res.Unchange {
			// Preserve the existing Plugin contract: changed content must be *T.
			// Buggy Plugin implementations still panic here, by design.
			content = retContent.(*T)
		}
	}
	return content, nil
}

func (m *Manager) Register(p Plugin) {
	if p.IsSupport(OpLogin) {
		m.loginPlugins = append(m.loginPlugins, p)
	}
	if p.IsSupport(OpNewProxy) {
		m.newProxyPlugins = append(m.newProxyPlugins, p)
	}
	if p.IsSupport(OpCloseProxy) {
		m.closeProxyPlugins = append(m.closeProxyPlugins, p)
	}
	if p.IsSupport(OpPing) {
		m.pingPlugins = append(m.pingPlugins, p)
	}
	if p.IsSupport(OpNewWorkConn) {
		m.newWorkConnPlugins = append(m.newWorkConnPlugins, p)
	}
	if p.IsSupport(OpNewUserConn) {
		m.newUserConnPlugins = append(m.newUserConnPlugins, p)
	}
}

func (m *Manager) Login(content *LoginContent) (*LoginContent, error) {
	return handleMutableContent(m.loginPlugins, OpLogin, content, pluginErrorLogWarn)
}

func (m *Manager) NewProxy(content *NewProxyContent) (*NewProxyContent, error) {
	return handleMutableContent(m.newProxyPlugins, OpNewProxy, content, pluginErrorLogWarn)
}

func (m *Manager) CloseProxy(content *CloseProxyContent) error {
	if len(m.closeProxyPlugins) == 0 {
		return nil
	}

	errs := make([]string, 0)
	ctx, xl := newPluginRequestContext()

	for _, p := range m.closeProxyPlugins {
		_, _, err := p.Handle(ctx, OpCloseProxy, *content)
		if err != nil {
			xl.Warnf("send CloseProxy request to plugin [%s] error: %v", p.Name(), err)
			errs = append(errs, fmt.Sprintf("[%s]: %v", p.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("send CloseProxy request to plugin errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (m *Manager) Ping(content *PingContent) (*PingContent, error) {
	return handleMutableContent(m.pingPlugins, OpPing, content, pluginErrorLogWarn)
}

func (m *Manager) NewWorkConn(content *NewWorkConnContent) (*NewWorkConnContent, error) {
	return handleMutableContent(m.newWorkConnPlugins, OpNewWorkConn, content, pluginErrorLogWarn)
}

func (m *Manager) NewUserConn(content *NewUserConnContent) (*NewUserConnContent, error) {
	// Preserve the pre-refactor log level for NewUserConn plugin errors.
	return handleMutableContent(m.newUserConnPlugins, OpNewUserConn, content, pluginErrorLogInfo)
}
