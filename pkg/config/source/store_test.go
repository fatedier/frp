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

package source

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestStoreSource_AddProxyAndVisitor_DoesNotApplyRuntimeDefaults(t *testing.T) {
	require := require.New(t)

	path := filepath.Join(t.TempDir(), "store.json")
	storeSource, err := NewStoreSource(StoreSourceConfig{Path: path})
	require.NoError(err)

	proxyCfg := &v1.TCPProxyConfig{}
	proxyCfg.Name = "proxy1"
	proxyCfg.Type = "tcp"
	proxyCfg.LocalPort = 10080

	visitorCfg := &v1.XTCPVisitorConfig{}
	visitorCfg.Name = "visitor1"
	visitorCfg.Type = "xtcp"
	visitorCfg.ServerName = "server1"
	visitorCfg.SecretKey = "secret"
	visitorCfg.BindPort = 10081

	err = storeSource.AddProxy(proxyCfg)
	require.NoError(err)
	err = storeSource.AddVisitor(visitorCfg)
	require.NoError(err)

	gotProxy := storeSource.GetProxy("proxy1")
	require.NotNil(gotProxy)
	require.Empty(gotProxy.GetBaseConfig().LocalIP)

	gotVisitor := storeSource.GetVisitor("visitor1")
	require.NotNil(gotVisitor)
	require.Empty(gotVisitor.GetBaseConfig().BindAddr)
	require.Empty(gotVisitor.(*v1.XTCPVisitorConfig).Protocol)
}

func TestStoreSource_LoadFromFile_DoesNotApplyRuntimeDefaults(t *testing.T) {
	require := require.New(t)

	path := filepath.Join(t.TempDir(), "store.json")

	proxyCfg := &v1.TCPProxyConfig{}
	proxyCfg.Name = "proxy1"
	proxyCfg.Type = "tcp"
	proxyCfg.LocalPort = 10080

	visitorCfg := &v1.XTCPVisitorConfig{}
	visitorCfg.Name = "visitor1"
	visitorCfg.Type = "xtcp"
	visitorCfg.ServerName = "server1"
	visitorCfg.SecretKey = "secret"
	visitorCfg.BindPort = 10081

	stored := storeData{
		Proxies:  []v1.TypedProxyConfig{{ProxyConfigurer: proxyCfg}},
		Visitors: []v1.TypedVisitorConfig{{VisitorConfigurer: visitorCfg}},
	}
	data, err := json.Marshal(stored)
	require.NoError(err)
	err = os.WriteFile(path, data, 0o600)
	require.NoError(err)

	storeSource, err := NewStoreSource(StoreSourceConfig{Path: path})
	require.NoError(err)

	gotProxy := storeSource.GetProxy("proxy1")
	require.NotNil(gotProxy)
	require.Empty(gotProxy.GetBaseConfig().LocalIP)

	gotVisitor := storeSource.GetVisitor("visitor1")
	require.NotNil(gotVisitor)
	require.Empty(gotVisitor.GetBaseConfig().BindAddr)
	require.Empty(gotVisitor.(*v1.XTCPVisitorConfig).Protocol)
}
