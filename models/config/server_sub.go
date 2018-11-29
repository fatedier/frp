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
	"strings"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/utils/util"
)

// common config
type SubServerSectionConf struct {
	Section       string `json:"section"`

	Token         string `json:"token"`
	AllowPorts    map[int]struct{}
}

func GetDefaultSubServerConf() *SubServerSectionConf {
	return &SubServerSectionConf{
		Section:           "",
		Token:             "",
		AllowPorts:        make(map[int]struct{}),
	}
}

func UnmarshalSubServerConfFromIni(content string, section string) (cfg *SubServerSectionConf, err error) {
	cfg = GetDefaultSubServerConf()

	conf, err := ini.Load(strings.NewReader(content))
	if err != nil {
		err = fmt.Errorf("parse ini conf file error: %v", err)
		return nil, err
	}

	cfg.Section = section

	cfg.Token, _ = conf.Get(section, "token")

	if allowPortsStr, ok := conf.Get(section, "allow_ports"); ok {
		// e.g. 1000-2000,2001,2002,3000-4000
		ports, errRet := util.ParseRangeNumbers(allowPortsStr)
		if errRet != nil {
			err = fmt.Errorf("Parse ini[%s] conf error: allow_ports: %v", section, errRet)
			return
		}

		for _, port := range ports {
			cfg.AllowPorts[int(port)] = struct{}{}
		}
	}

	return
}
