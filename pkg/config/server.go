// Copyright 2020 The frp Authors
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

package config

import (
	"fmt"
	"strings"

	"github.com/fatedier/frp/pkg/auth"
	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/util"

	"gopkg.in/ini.v1"
)

// GetDefaultServerConf returns a server configuration with reasonable
// defaults.
func GetDefaultServerConf() ServerCommonConf {
	return ServerCommonConf{
		ServerConfig:           auth.GetDefaultServerConf(),
		BindAddr:               "0.0.0.0",
		BindPort:               7000,
		BindUDPPort:            0,
		KCPBindPort:            0,
		ProxyBindAddr:          "0.0.0.0",
		VhostHTTPPort:          0,
		VhostHTTPSPort:         0,
		TCPMuxHTTPConnectPort:  0,
		VhostHTTPTimeout:       60,
		DashboardAddr:          "0.0.0.0",
		DashboardPort:          0,
		DashboardUser:          "admin",
		DashboardPwd:           "admin",
		EnablePrometheus:       false,
		AssetsDir:              "",
		LogFile:                "console",
		LogWay:                 "console",
		LogLevel:               "info",
		LogMaxDays:             3,
		DisableLogColor:        false,
		DetailedErrorsToClient: true,
		SubDomainHost:          "",
		TCPMux:                 true,
		AllowPorts:             make(map[int]struct{}),
		MaxPoolCount:           5,
		MaxPortsPerClient:      0,
		TLSOnly:                false,
		TLSCertFile:            "",
		TLSKeyFile:             "",
		TLSTrustedCaFile:       "",
		HeartbeatTimeout:       90,
		UserConnTimeout:        10,
		Custom404Page:          "",
		HTTPPlugins:            make(map[string]plugin.HTTPPluginOptions),
		UDPPacketSize:          1500,
	}
}

func (cfg *ServerCommonConf) Check() error {
	return nil
}

func UnmarshalServerConfFromIni(source interface{}) (ServerCommonConf, error) {

	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ServerCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		// TODO: add error info
		return ServerCommonConf{}, err
	}

	common := GetDefaultServerConf()
	err = s.MapTo(&common)
	if err != nil {
		return ServerCommonConf{}, err
	}

	// allow_ports
	allowPortStr := s.Key("allow_ports").String()
	if allowPortStr != "" {
		allowPorts, err := util.ParseRangeNumbers(allowPortStr)
		if err != nil {
			return ServerCommonConf{}, fmt.Errorf("Parse conf error: allow_ports: %v", err)
		}
		for _, port := range allowPorts {
			common.AllowPorts[int(port)] = struct{}{}
		}
	}

	// plugin.xxx
	pluginOpts := make(map[string]plugin.HTTPPluginOptions)
	for _, section := range f.Sections() {
		name := section.Name()
		if !strings.HasPrefix(name, "plugin.") {
			continue
		}

		opt, err := loadHTTPPluginOpt(section)
		if err != nil {
			return ServerCommonConf{}, err
		}

		pluginOpts[opt.Name] = *opt
	}
	common.HTTPPlugins = pluginOpts

	return common, nil
}

func loadHTTPPluginOpt(section *ini.Section) (*plugin.HTTPPluginOptions, error) {
	name := strings.TrimSpace(strings.TrimPrefix(section.Name(), "plugin."))

	opt := new(plugin.HTTPPluginOptions)
	err := section.MapTo(opt)
	if err != nil {
		return nil, err
	}

	opt.Name = name

	return opt, nil
}
