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

	"github.com/samber/lo"
	"gopkg.in/ini.v1"

	"github.com/fatedier/frp/pkg/consts"
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
	// GetBaseConfig returns the base config of visitor.
	GetBaseConfig() *BaseVisitorConf
	// UnmarshalFromIni unmarshals config from ini.
	UnmarshalFromIni(prefix string, name string, section *ini.Section) error
	// Validate validates config.
	Validate() error
}

type BaseVisitorConf struct {
	ProxyName      string `ini:"name" json:"name"`
	ProxyType      string `ini:"type" json:"type"`
	UseEncryption  bool   `ini:"use_encryption" json:"use_encryption"`
	UseCompression bool   `ini:"use_compression" json:"use_compression"`
	Role           string `ini:"role" json:"role"`
	Sk             string `ini:"sk" json:"sk"`
	// if the server user is not set, it defaults to the current user
	ServerUser string `ini:"server_user" json:"server_user"`
	ServerName string `ini:"server_name" json:"server_name"`
	BindAddr   string `ini:"bind_addr" json:"bind_addr"`
	// BindPort is the port that visitor listens on.
	// It can be less than 0, it means don't bind to the port and only receive connections redirected from
	// other visitors. (This is not supported for SUDP now)
	BindPort int `ini:"bind_port" json:"bind_port"`
}

type SUDPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

type STCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`
}

type XTCPVisitorConf struct {
	BaseVisitorConf `ini:",extends"`

	Protocol          string `ini:"protocol" json:"protocol,omitempty"`
	KeepTunnelOpen    bool   `ini:"keep_tunnel_open" json:"keep_tunnel_open,omitempty"`
	MaxRetriesAnHour  int    `ini:"max_retries_an_hour" json:"max_retries_an_hour,omitempty"`
	MinRetryInterval  int    `ini:"min_retry_interval" json:"min_retry_interval,omitempty"`
	FallbackTo        string `ini:"fallback_to" json:"fallback_to,omitempty"`
	FallbackTimeoutMs int    `ini:"fallback_timeout_ms" json:"fallback_timeout_ms,omitempty"`
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
		return nil, fmt.Errorf("type shouldn't be empty")
	}

	conf := DefaultVisitorConf(visitorType)
	if conf == nil {
		return nil, fmt.Errorf("type [%s] error", visitorType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, fmt.Errorf("type [%s] error", visitorType)
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Base
func (cfg *BaseVisitorConf) GetBaseConfig() *BaseVisitorConf {
	return cfg
}

func (cfg *BaseVisitorConf) validate() (err error) {
	if cfg.Role != "visitor" {
		err = fmt.Errorf("invalid role")
		return
	}
	if cfg.BindAddr == "" {
		err = fmt.Errorf("bind_addr shouldn't be empty")
		return
	}
	// BindPort can be less than 0, it means don't bind to the port and only receive connections redirected from
	// other visitors
	if cfg.BindPort == 0 {
		err = fmt.Errorf("bind_port is required")
		return
	}
	return
}

func (cfg *BaseVisitorConf) unmarshalFromIni(prefix string, name string, section *ini.Section) error {
	_ = section

	// Custom decoration after basic unmarshal:
	// proxy name
	cfg.ProxyName = prefix + name

	// server_name
	if cfg.ServerUser == "" {
		cfg.ServerName = prefix + cfg.ServerName
	} else {
		cfg.ServerName = cfg.ServerUser + "." + cfg.ServerName
	}

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

	err = cfg.GetBaseConfig().unmarshalFromIni(prefix, name, section)
	if err != nil {
		return err
	}
	return nil
}

// SUDP
var _ VisitorConf = &SUDPVisitorConf{}

func (cfg *SUDPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists

	return
}

func (cfg *SUDPVisitorConf) Validate() (err error) {
	if err = cfg.BaseVisitorConf.validate(); err != nil {
		return
	}

	// Add custom logic validate, if exists

	return
}

// STCP
var _ VisitorConf = &STCPVisitorConf{}

func (cfg *STCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists

	return
}

func (cfg *STCPVisitorConf) Validate() (err error) {
	if err = cfg.BaseVisitorConf.validate(); err != nil {
		return
	}

	// Add custom logic validate, if exists

	return
}

// XTCP
var _ VisitorConf = &XTCPVisitorConf{}

func (cfg *XTCPVisitorConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) (err error) {
	err = preVisitorUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return
	}

	// Add custom logic unmarshal, if exists
	if cfg.Protocol == "" {
		cfg.Protocol = "quic"
	}
	if cfg.MaxRetriesAnHour <= 0 {
		cfg.MaxRetriesAnHour = 8
	}
	if cfg.MinRetryInterval <= 0 {
		cfg.MinRetryInterval = 90
	}
	if cfg.FallbackTimeoutMs <= 0 {
		cfg.FallbackTimeoutMs = 1000
	}
	return
}

func (cfg *XTCPVisitorConf) Validate() (err error) {
	if err = cfg.BaseVisitorConf.validate(); err != nil {
		return
	}

	// Add custom logic validate, if exists
	if !lo.Contains([]string{"", "kcp", "quic"}, cfg.Protocol) {
		return fmt.Errorf("protocol should be 'kcp' or 'quic'")
	}
	return
}
