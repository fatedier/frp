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

	"github.com/fatedier/frp/pkg/consts"

	"gopkg.in/ini.v1"
)

// Visitor
var (
	visitorConfTypeMap = map[string]reflect.Type{
		consts.STCPProxy: reflect.TypeOf(STCPVisitorConf{}),
		consts.XTCPProxy: reflect.TypeOf(XTCPVisitorConf{}),
		consts.SUDPProxy: reflect.TypeOf(SUDPVisitorConf{}),
	}
)

type VisitorConf interface {
	GetBaseInfo() *BaseVisitorConf
	Compare(cmp VisitorConf) bool
	UnmarshalFromIni(prefix string, name string, section *ini.Section) error
	Check() error
}

type BaseVisitorConf struct {
	ProxyName      string `ini:"name" json:"name"`
	ProxyType      string `ini:"type" json:"type"`
	UseEncryption  bool   `ini:"use_encryption" json:"use_encryption"`
	UseCompression bool   `ini:"use_compression" json:"use_compression"`
	Role           string `ini:"role" json:"role"`
	Sk             string `ini:"sk" json:"sk"`
	ServerName     string `ini:"server_name" json:"server_name"`
	BindAddr       string `ini:"bind_addr" json:"bind_addr"`
	BindPort       int    `ini:"bind_port" json:"bind_port"`
}

type SUDPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

type STCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

type XTCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

// DefaultVisitorConf creates a empty VisitorConf object by visitorType.
// If visitorType doesn't exist, return nil.
func DefaultVisitorConf(visitorType string) VisitorConf {
	v, ok := visitorConfTypeMap[visitorType]
	if !ok {
		return nil
	}

	return reflect.New(v).Interface().(VisitorConf)
}

// Visitor loaded from ini
func NewVisitorConfFromIni(prefix string, name string, section *ini.Section) (VisitorConf, error) {
	// section.Key: if key not exists, section will set it with default value.
	visitorType := section.Key("type").String()

	if visitorType == "" {
		return nil, fmt.Errorf("visitor [%s] type shouldn't be empty", name)
	}

	conf := DefaultVisitorConf(visitorType)
	if conf == nil {
		return nil, fmt.Errorf("visitor [%s] type [%s] error", name, visitorType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, fmt.Errorf("visitor [%s] type [%s] error", name, visitorType)
	}

	if err := conf.Check(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Base
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

func (cfg *BaseVisitorConf) unmarshalFromIni(prefix string, name string, section *ini.Section) error {

	// Custom decoration after basic unmarshal:
	// proxy name
	cfg.ProxyName = prefix + name

	// server_name
	cfg.ServerName = prefix + cfg.ServerName

	// bind_addr
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}

	return nil
}

func preVisitorUnmarshalFromIni(cfg VisitorConf, prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.GetBaseInfo().unmarshalFromIni(prefix, name, section)
	if err != nil {
		return err
	}

	return nil
}

// SUDP
var _ VisitorConf = &SUDPVisitorConf{}

func (cfg *SUDPVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*SUDPVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (cfg *SUDPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists

	return
}

func (cfg *SUDPVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}

	// Add custom logic validate, if exists

	return
}

// STCP
var _ VisitorConf = &STCPVisitorConf{}

func (cfg *STCPVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*STCPVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (cfg *STCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists

	return
}

func (cfg *STCPVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}

	// Add custom logic validate, if exists

	return
}

// XTCP
var _ VisitorConf = &XTCPVisitorConf{}

func (cfg *XTCPVisitorConf) Compare(cmp VisitorConf) bool {
	cmpConf, ok := cmp.(*XTCPVisitorConf)
	if !ok {
		return false
	}

	if !cfg.BaseVisitorConf.compare(&cmpConf.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (cfg *XTCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists

	return
}

func (cfg *XTCPVisitorConf) Check() (err error) {
	if err = cfg.BaseVisitorConf.check(); err != nil {
		return
	}

	// Add custom logic validate, if exists

	return
}
