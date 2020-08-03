// Copyright 2018 fatedier, fatedier@gmail.com
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
	"reflect"
	"strconv"

	"github.com/fatedier/frp/models/consts"

	ini "github.com/vaughan0/go-ini"
)

var (
	visitorConfTypeMap map[string]reflect.Type
)

func init() {
	visitorConfTypeMap = make(map[string]reflect.Type)
	visitorConfTypeMap[consts.StcpProxy] = reflect.TypeOf(StcpVisitorConf{})
	visitorConfTypeMap[consts.XtcpProxy] = reflect.TypeOf(XtcpVisitorConf{})
	visitorConfTypeMap[consts.SudpProxy] = reflect.TypeOf(SudpVisitorConf{})
}

type VisitorConf interface {
	GetBaseInfo() *BaseVisitorConf
	Compare(cmp VisitorConf) bool
	UnmarshalFromIni(prefix string, name string, section ini.Section) error
	Check() error
}

func NewVisitorConfByType(cfgType string) VisitorConf {
	v, ok := visitorConfTypeMap[cfgType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(VisitorConf)
	return cfg
}

func NewVisitorConfFromIni(prefix string, name string, section ini.Section) (cfg VisitorConf, err error) {
	cfgType := section["type"]
	if cfgType == "" {
		err = fmt.Errorf("visitor [%s] type shouldn't be empty", name)
		return
	}
	cfg = NewVisitorConfByType(cfgType)
	if cfg == nil {
		err = fmt.Errorf("visitor [%s] type [%s] error", name, cfgType)
		return
	}
	if err = cfg.UnmarshalFromIni(prefix, name, section); err != nil {
		return
	}
	if err = cfg.Check(); err != nil {
		return
	}
	return
}

type BaseVisitorConf struct {
	ProxyName      string `json:"proxy_name"`
	ProxyType      string `json:"proxy_type"`
	UseEncryption  bool   `json:"use_encryption"`
	UseCompression bool   `json:"use_compression"`
	Role           string `json:"role"`
	Sk             string `json:"sk"`
	ServerName     string `json:"server_name"`
	BindAddr       string `json:"bind_addr"`
	BindPort       int    `json:"bind_port"`
}

func (cfg *BaseVisitorConf) GetBaseInfo() *BaseVisitorConf {
	return cfg
}

func (cfg *BaseVisitorConf) compare(cmp *BaseVisitorConf) bool {
	if cfg.ProxyName != cmp.ProxyName ||
		cfg.ProxyType != cmp.ProxyType ||
		cfg.UseEncryption != cmp.UseEncryption ||
		cfg.UseCompression != cmp.UseCompression ||
		cfg.Role != cmp.Role ||
		cfg.Sk != cmp.Sk ||
		cfg.ServerName != cmp.ServerName ||
		cfg.BindAddr != cmp.BindAddr ||
		cfg.BindPort != cmp.BindPort {
		return false
	}
	return true
}

func (cfg *BaseVisitorConf) check() (err error) {
	if cfg.Role != "visitor" {
		err = fmt.Errorf("invalid role")
		return
	}
	if cfg.BindAddr == "" {
		err = fmt.Errorf("bind_addr shouldn't be empty")
		return
	}
	if cfg.BindPort <= 0 {
		err = fmt.Errorf("bind_port is required")
		return
	}
	return
}

func (cfg *BaseVisitorConf) UnmarshalFromIni(prefix string, name string, section ini.Section) (err error) {
	var (
		tmpStr string
		ok     bool
	)
	cfg.ProxyName = prefix + name
	cfg.ProxyType = section["type"]

	if tmpStr, ok = section["use_encryption"]; ok && tmpStr == "true" {
		cfg.UseEncryption = true
	}
	if tmpStr, ok = section["use_compression"]; ok && tmpStr == "true" {
		cfg.UseCompression = true
	}

	cfg.Role = section["role"]
	if cfg.Role != "visitor" {
		return fmt.Errorf("Parse conf error: proxy [%s] incorrect role [%s]", name, cfg.Role)
	}
	cfg.Sk = section["sk"]
	cfg.ServerName = prefix + section["server_name"]
	if cfg.BindAddr = section["bind_addr"]; cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}

	if tmpStr, ok = section["bind_port"]; ok {
		if cfg.BindPort, err = strconv.Atoi(tmpStr); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] bind_port incorrect", name)
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] bind_port not found", name)
	}
	return nil
}

type SudpVisitorConf struct {
	BaseVisitorConf
}

func (cfg *SudpVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*SudpVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}
	return true
}

func (cfg *SudpVisitorConf) UnmarshalFromIni(prefix string, name string, section ini.Section) (err error) {
	if err = cfg.BaseVisitorConf.UnmarshalFromIni(prefix, name, section); err != nil {
		return
	}
	return
}

func (cfg *SudpVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}
	return
}

type StcpVisitorConf struct {
	BaseVisitorConf
}

func (cfg *StcpVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*StcpVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}
	return true
}

func (cfg *StcpVisitorConf) UnmarshalFromIni(prefix string, name string, section ini.Section) (err error) {
	if err = cfg.BaseVisitorConf.UnmarshalFromIni(prefix, name, section); err != nil {
		return
	}
	return
}

func (cfg *StcpVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}
	return
}

type XtcpVisitorConf struct {
	BaseVisitorConf
}

func (cfg *XtcpVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*XtcpVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}
	return true
}

func (cfg *XtcpVisitorConf) UnmarshalFromIni(prefix string, name string, section ini.Section) (err error) {
	if err = cfg.BaseVisitorConf.UnmarshalFromIni(prefix, name, section); err != nil {
		return
	}
	return
}

func (cfg *XtcpVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}
	return
}
