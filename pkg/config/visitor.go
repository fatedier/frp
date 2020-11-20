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

	"gopkg.in/ini.v1"
)

// Visitor Conf Loader
// DefaultVisitorConf creates a empty VisitorConf object by visitorType.
// If visitorType doesn't exist, return nil.
func DefaultVisitorConf(visitorType string) VisitorConf {
	v, ok := VisitorConfTypeMap[visitorType]
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
func (c *BaseVisitorConf) GetBaseInfo() *BaseVisitorConf {
	return c
}

func (c *BaseVisitorConf) compare(cmp *BaseVisitorConf) bool {
	if c.ProxyName != cmp.ProxyName ||
		c.ProxyType != cmp.ProxyType ||
		c.UseEncryption != cmp.UseEncryption ||
		c.UseCompression != cmp.UseCompression ||
		c.Role != cmp.Role ||
		c.Sk != cmp.Sk ||
		c.ServerName != cmp.ServerName ||
		c.BindAddr != cmp.BindAddr ||
		c.BindPort != cmp.BindPort {
		return false
	}
	return true
}

func (c *BaseVisitorConf) check() (err error) {
	if c.Role != "visitor" {
		err = fmt.Errorf("invalid role")
		return
	}
	if c.BindAddr == "" {
		err = fmt.Errorf("bind_addr shouldn't be empty")
		return
	}
	if c.BindPort <= 0 {
		err = fmt.Errorf("bind_port is required")
		return
	}
	return
}

func (c *BaseVisitorConf) decorate(prefix string, name string, section *ini.Section) error {

	// proxy name
	c.ProxyName = prefix + name

	// server_name
	c.ServerName = prefix + c.ServerName

	// bind_addr
	if c.BindAddr == "" {
		c.BindAddr = "127.0.0.1"
	}

	return nil
}

// STCP
var _ VisitorConf = &STCPVisitorConf{}

func (c *STCPVisitorConf) Compare(conf VisitorConf) bool {
	cmp, ok := conf.(*STCPVisitorConf)
	if !ok {
		return false
	}

	if !c.BaseVisitorConf.compare(&cmp.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (c *STCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseVisitorConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal, if exists

	return nil
}

func (cfg *STCPVisitorConf) Check() error {
	if err := cfg.BaseVisitorConf.check(); err != nil {
		return err
	}

	// Add custom logic validate, if exists

	return nil
}

// SUDP
var _ VisitorConf = &SUDPVisitorConf{}

func (c *SUDPVisitorConf) Compare(conf VisitorConf) bool {
	cmp, ok := conf.(*SUDPVisitorConf)
	if !ok {
		return false
	}

	if !c.BaseVisitorConf.compare(&cmp.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (c *SUDPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseVisitorConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal, if exists

	return nil
}

func (cfg *SUDPVisitorConf) Check() error {
	if err := cfg.BaseVisitorConf.check(); err != nil {
		return err
	}

	// Add custom logic validate, if exists

	return nil
}

// XTCP
var _ VisitorConf = &XTCPVisitorConf{}

func (c *XTCPVisitorConf) Compare(conf VisitorConf) bool {
	cmp, ok := conf.(*XTCPVisitorConf)
	if !ok {
		return false
	}

	if !c.BaseVisitorConf.compare(&cmp.BaseVisitorConf) {
		return false
	}

	// Add custom login equal, if exists

	return true
}

func (c *XTCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseVisitorConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal, if exists

	return nil
}

func (cfg *XTCPVisitorConf) Check() error {
	if err := cfg.BaseVisitorConf.check(); err != nil {
		return err
	}

	// Add custom logic validate, if exists

	return nil
}
