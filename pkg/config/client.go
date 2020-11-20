// Copyright 2016 fatedier, fatedier@gmail.com
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
	"os"
	"strings"

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/util/util"

	"gopkg.in/ini.v1"
)

// GetDefaultClientConf returns a client configuration with default values.
func DefaultClientConf() ClientCommonConf {
	return ClientCommonConf{
		ClientConfig:      auth.GetDefaultClientConf(),
		ServerAddr:        "0.0.0.0",
		ServerPort:        7000,
		HTTPProxy:         os.Getenv("http_proxy"),
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		DisableLogColor:   false,
		AdminAddr:         "127.0.0.1",
		AdminPort:         0,
		AdminUser:         "",
		AdminPwd:          "",
		AssetsDir:         "",
		PoolCount:         1,
		TCPMux:            true,
		User:              "",
		DNSServer:         "",
		LoginFailExit:     true,
		Start:             make([]string, 0),
		Protocol:          "tcp",
		TLSEnable:         false,
		TLSCertFile:       "",
		TLSKeyFile:        "",
		TLSTrustedCaFile:  "",
		HeartbeatInterval: 30,
		HeartbeatTimeout:  90,
		Metas:             make(map[string]string),
		UDPPacketSize:     1500,
	}
}

func (cfg *ClientCommonConf) Check() error {
	if cfg.HeartbeatInterval <= 0 {
		return fmt.Errorf("Parse conf error: invalid heartbeat_interval")
	}

	if cfg.HeartbeatTimeout < cfg.HeartbeatInterval {
		return fmt.Errorf("Parse conf error: invalid heartbeat_timeout, heartbeat_timeout is less than heartbeat_interval")
	}

	if cfg.TLSEnable == false {
		if cfg.TLSCertFile != "" {
			fmt.Println("WARNING! tls_cert_file is invalid when tls_enable is false")
		}

		if cfg.TLSKeyFile != "" {
			fmt.Println("WARNING! tls_key_file is invalid when tls_enable is false")
		}

		if cfg.TLSTrustedCaFile != "" {
			fmt.Println("WARNING! tls_trusted_ca_file is invalid when tls_enable is false")
		}
	}

	return nil
}

// Supported sources including: string(file path), []byte, Reader interface.
func LoadClientCommonConf(source interface{}) (ClientCommonConf, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ClientCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		// TODO: add error info
		return ClientCommonConf{}, err
	}

	common := DefaultClientConf()
	err = s.MapTo(&common)
	if err != nil {
		return ClientCommonConf{}, err
	}

	common.Metas = GetMapWithoutPrefix(s.KeysHash(), "meta_")

	return common, nil
}

// if len(startProxy) is 0, start all
// otherwise just start proxies in startProxy map
func LoadClientBasicConf(
	prefix string,
	source interface{},
	start []string,
) (map[string]ProxyConf, map[string]VisitorConf, error) {

	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return nil, nil, err
	}

	proxyConfs := make(map[string]ProxyConf)
	visitorConfs := make(map[string]VisitorConf)

	if prefix != "" {
		prefix += "."
	}

	startProxy := make(map[string]struct{})
	for _, s := range start {
		startProxy[s] = struct{}{}
	}

	startAll := true
	if len(startProxy) > 0 {
		startAll = false
	}

	// Build template sections from range section And append to ini.File.
	rangeSections := make([]*ini.Section, 0)
	for _, section := range f.Sections() {

		if !strings.HasPrefix(section.Name(), "range:") {
			continue
		}

		rangeSections = append(rangeSections, section)
	}

	for _, section := range rangeSections {
		err = appendTemplates(f, section)
		if err != nil {
			return nil, nil, err
		}
	}

	for _, section := range f.Sections() {
		name := section.Name()

		if name == ini.DefaultSection || name == "common" || strings.HasPrefix(name, "range:") {
			continue
		}

		_, shouldStart := startProxy[name]
		if !startAll && !shouldStart {
			continue
		}

		roleType := section.Key("role").String()
		if roleType == "" {
			roleType = "server"
		}

		switch roleType {
		case "server":
			newConf, newErr := NewProxyConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, fmt.Errorf("fail to parse section[%s], err: %v", name, newErr)
			}
			proxyConfs[prefix+name] = newConf
		case "visitor":
			newConf, newErr := NewVisitorConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, newErr
			}
			visitorConfs[prefix+name] = newConf
		default:
			return nil, nil, fmt.Errorf("section[%s] role should be 'server' or 'visitor'", name)
		}
	}
	return proxyConfs, visitorConfs, nil
}

func appendTemplates(f *ini.File, section *ini.Section) error {

	// Validation
	localPortStr := section.Key("local_port").String()
	remotePortStr := section.Key("remote_port").String()
	if localPortStr == "" || remotePortStr == "" {
		return fmt.Errorf("local_port or remote_port is empty")
	}

	localPorts, err := util.ParseRangeNumbers(localPortStr)
	if err != nil {
		return err
	}

	remotePorts, err := util.ParseRangeNumbers(remotePortStr)
	if err != nil {
		return err
	}

	if len(localPorts) != len(remotePorts) {
		return fmt.Errorf("range section [%s] local ports number should be same with remote ports number", section.Name())
	}

	if len(localPorts) == 0 {
		return fmt.Errorf("range section [%s] local_port and remote_port is necessary", section.Name())
	}

	// Templates
	prefix := strings.TrimSpace(strings.TrimPrefix(section.Name(), "range:"))

	for i := range localPorts {
		tmpname := fmt.Sprintf("%s_%d", prefix, i)

		tmpsection, err := f.NewSection(tmpname)
		if err != nil {
			return err
		}

		copySection(section, tmpsection)
		tmpsection.NewKey("local_port", fmt.Sprintf("%d", localPorts[i]))
		tmpsection.NewKey("remote_port", fmt.Sprintf("%d", remotePorts[i]))
	}

	return nil
}

func copySection(source, target *ini.Section) {
	for key, value := range source.KeysHash() {
		target.NewKey(key, value)
	}
}
