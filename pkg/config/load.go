// Copyright 2023 The frp Authors
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
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
	"gopkg.in/ini.v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/fatedier/frp/pkg/config/legacy"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

var glbEnvs map[string]string

func init() {
	glbEnvs = make(map[string]string)
	envs := os.Environ()
	for _, env := range envs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}
		glbEnvs[pair[0]] = pair[1]
	}
}

type Values struct {
	Envs map[string]string // environment vars
}

func GetValues() *Values {
	return &Values{
		Envs: glbEnvs,
	}
}

func DetectLegacyINIFormat(content []byte) bool {
	f, err := ini.Load(content)
	if err != nil {
		return false
	}
	if _, err := f.GetSection("common"); err == nil {
		return true
	}
	return false
}

func DetectLegacyINIFormatFromFile(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return DetectLegacyINIFormat(b)
}

func RenderWithTemplate(in []byte, values *Values) ([]byte, error) {
	tmpl, err := template.New("frp").Funcs(template.FuncMap{
		"parseNumberRange":     parseNumberRange,
		"parseNumberRangePair": parseNumberRangePair,
	}).Parse(string(in))
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBufferString("")
	if err := tmpl.Execute(buffer, values); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func LoadFileContentWithTemplate(path string, values *Values) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return RenderWithTemplate(b, values)
}

func LoadConfigureFromFile(path string, c any, strict bool) error {
	content, err := LoadFileContentWithTemplate(path, GetValues())
	if err != nil {
		return err
	}
	return LoadConfigure(content, c, strict)
}

// LoadConfigure loads configuration from bytes and unmarshal into c.
// Now it supports json, yaml and toml format.
func LoadConfigure(b []byte, c any, strict bool) error {
	v1.DisallowUnknownFieldsMu.Lock()
	defer v1.DisallowUnknownFieldsMu.Unlock()
	v1.DisallowUnknownFields = strict

	var tomlObj interface{}
	// Try to unmarshal as TOML first; swallow errors from that (assume it's not valid TOML).
	if err := toml.Unmarshal(b, &tomlObj); err == nil {
		b, err = json.Marshal(&tomlObj)
		if err != nil {
			return err
		}
	}
	// If the buffer smells like JSON (first non-whitespace character is '{'), unmarshal as JSON directly.
	if yaml.IsJSONBuffer(b) {
		decoder := json.NewDecoder(bytes.NewBuffer(b))
		if strict {
			decoder.DisallowUnknownFields()
		}
		return decoder.Decode(c)
	}
	// It wasn't JSON. Unmarshal as YAML.
	if strict {
		return yaml.UnmarshalStrict(b, c)
	}
	return yaml.Unmarshal(b, c)
}

func NewProxyConfigurerFromMsg(m *msg.NewProxy, serverCfg *v1.ServerConfig) (v1.ProxyConfigurer, error) {
	m.ProxyType = util.EmptyOr(m.ProxyType, string(v1.ProxyTypeTCP))

	configurer := v1.NewProxyConfigurerByType(v1.ProxyType(m.ProxyType))
	if configurer == nil {
		return nil, fmt.Errorf("unknown proxy type: %s", m.ProxyType)
	}

	configurer.UnmarshalFromMsg(m)
	configurer.Complete("")

	if err := validation.ValidateProxyConfigurerForServer(configurer, serverCfg); err != nil {
		return nil, err
	}
	return configurer, nil
}

func LoadServerConfig(path string, strict bool) (*v1.ServerConfig, bool, error) {
	var (
		svrCfg         *v1.ServerConfig
		isLegacyFormat bool
	)
	// detect legacy ini format
	if DetectLegacyINIFormatFromFile(path) {
		content, err := legacy.GetRenderedConfFromFile(path)
		if err != nil {
			return nil, true, err
		}
		legacyCfg, err := legacy.UnmarshalServerConfFromIni(content)
		if err != nil {
			return nil, true, err
		}
		svrCfg = legacy.Convert_ServerCommonConf_To_v1(&legacyCfg)
		isLegacyFormat = true
	} else {
		svrCfg = &v1.ServerConfig{}
		if err := LoadConfigureFromFile(path, svrCfg, strict); err != nil {
			return nil, false, err
		}
	}
	if svrCfg != nil {
		svrCfg.Complete()
	}
	return svrCfg, isLegacyFormat, nil
}

func LoadClientConfig(path string, strict bool) (
	*v1.ClientCommonConfig,
	[]v1.ProxyConfigurer,
	[]v1.VisitorConfigurer,
	bool, error,
) {
	var (
		cliCfg         *v1.ClientCommonConfig
		proxyCfgs      = make([]v1.ProxyConfigurer, 0)
		visitorCfgs    = make([]v1.VisitorConfigurer, 0)
		isLegacyFormat bool
	)

	if DetectLegacyINIFormatFromFile(path) {
		legacyCommon, legacyProxyCfgs, legacyVisitorCfgs, err := legacy.ParseClientConfig(path)
		if err != nil {
			return nil, nil, nil, true, err
		}
		cliCfg = legacy.Convert_ClientCommonConf_To_v1(&legacyCommon)
		for _, c := range legacyProxyCfgs {
			proxyCfgs = append(proxyCfgs, legacy.Convert_ProxyConf_To_v1(c))
		}
		for _, c := range legacyVisitorCfgs {
			visitorCfgs = append(visitorCfgs, legacy.Convert_VisitorConf_To_v1(c))
		}
		isLegacyFormat = true
	} else {
		allCfg := v1.ClientConfig{}
		if err := LoadConfigureFromFile(path, &allCfg, strict); err != nil {
			return nil, nil, nil, false, err
		}
		cliCfg = &allCfg.ClientCommonConfig
		for _, c := range allCfg.Proxies {
			proxyCfgs = append(proxyCfgs, c.ProxyConfigurer)
		}
		for _, c := range allCfg.Visitors {
			visitorCfgs = append(visitorCfgs, c.VisitorConfigurer)
		}
	}

	// Load additional config from includes.
	// legacy ini format already handle this in ParseClientConfig.
	if len(cliCfg.IncludeConfigFiles) > 0 && !isLegacyFormat {
		extProxyCfgs, extVisitorCfgs, err := LoadAdditionalClientConfigs(cliCfg.IncludeConfigFiles, isLegacyFormat, strict)
		if err != nil {
			return nil, nil, nil, isLegacyFormat, err
		}
		proxyCfgs = append(proxyCfgs, extProxyCfgs...)
		visitorCfgs = append(visitorCfgs, extVisitorCfgs...)
	}

	// Filter by start
	if len(cliCfg.Start) > 0 {
		startSet := sets.New(cliCfg.Start...)
		proxyCfgs = lo.Filter(proxyCfgs, func(c v1.ProxyConfigurer, _ int) bool {
			return startSet.Has(c.GetBaseConfig().Name)
		})
		visitorCfgs = lo.Filter(visitorCfgs, func(c v1.VisitorConfigurer, _ int) bool {
			return startSet.Has(c.GetBaseConfig().Name)
		})
	}

	if cliCfg != nil {
		cliCfg.Complete()
	}
	for _, c := range proxyCfgs {
		c.Complete(cliCfg.User)
	}
	for _, c := range visitorCfgs {
		c.Complete(cliCfg)
	}
	return cliCfg, proxyCfgs, visitorCfgs, isLegacyFormat, nil
}

func LoadAdditionalClientConfigs(paths []string, isLegacyFormat bool, strict bool) ([]v1.ProxyConfigurer, []v1.VisitorConfigurer, error) {
	proxyCfgs := make([]v1.ProxyConfigurer, 0)
	visitorCfgs := make([]v1.VisitorConfigurer, 0)
	for _, path := range paths {
		absDir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return nil, nil, err
		}
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return nil, nil, err
		}
		files, err := os.ReadDir(absDir)
		if err != nil {
			return nil, nil, err
		}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			absFile := filepath.Join(absDir, fi.Name())
			if matched, _ := filepath.Match(filepath.Join(absDir, filepath.Base(path)), absFile); matched {
				// support yaml/json/toml
				cfg := v1.ClientConfig{}
				if err := LoadConfigureFromFile(absFile, &cfg, strict); err != nil {
					return nil, nil, fmt.Errorf("load additional config from %s error: %v", absFile, err)
				}
				for _, c := range cfg.Proxies {
					proxyCfgs = append(proxyCfgs, c.ProxyConfigurer)
				}
				for _, c := range cfg.Visitors {
					visitorCfgs = append(visitorCfgs, c.VisitorConfigurer)
				}
			}
		}
	}
	return proxyCfgs, visitorCfgs, nil
}
