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

package legacy

import (
	"fmt"
	"reflect"

	"gopkg.in/ini.v1"

	"github.com/fatedier/frp/pkg/config/types"
)

type ProxyType string

const (
	ProxyTypeTCP    ProxyType = "tcp"
	ProxyTypeUDP    ProxyType = "udp"
	ProxyTypeTCPMUX ProxyType = "tcpmux"
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSTCP   ProxyType = "stcp"
	ProxyTypeXTCP   ProxyType = "xtcp"
	ProxyTypeSUDP   ProxyType = "sudp"
)

// Proxy
var (
	proxyConfTypeMap = map[ProxyType]reflect.Type{
		ProxyTypeTCP:    reflect.TypeOf(TCPProxyConf{}),
		ProxyTypeUDP:    reflect.TypeOf(UDPProxyConf{}),
		ProxyTypeTCPMUX: reflect.TypeOf(TCPMuxProxyConf{}),
		ProxyTypeHTTP:   reflect.TypeOf(HTTPProxyConf{}),
		ProxyTypeHTTPS:  reflect.TypeOf(HTTPSProxyConf{}),
		ProxyTypeSTCP:   reflect.TypeOf(STCPProxyConf{}),
		ProxyTypeXTCP:   reflect.TypeOf(XTCPProxyConf{}),
		ProxyTypeSUDP:   reflect.TypeOf(SUDPProxyConf{}),
	}
)

type ProxyConf interface {
	// GetBaseConfig returns the BaseProxyConf for this config.
	GetBaseConfig() *BaseProxyConf
	// UnmarshalFromIni unmarshals a ini.Section into this config. This function
	// will be called on the frpc side.
	UnmarshalFromIni(string, string, *ini.Section) error
}

func NewConfByType(proxyType ProxyType) ProxyConf {
	v, ok := proxyConfTypeMap[proxyType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(ProxyConf)
	return cfg
}

// Proxy Conf Loader
// DefaultProxyConf creates a empty ProxyConf object by proxyType.
// If proxyType doesn't exist, return nil.
func DefaultProxyConf(proxyType ProxyType) ProxyConf {
	return NewConfByType(proxyType)
}

// Proxy loaded from ini
func NewProxyConfFromIni(prefix, name string, section *ini.Section) (ProxyConf, error) {
	// section.Key: if key not exists, section will set it with default value.
	proxyType := ProxyType(section.Key("type").String())
	if proxyType == "" {
		proxyType = ProxyTypeTCP
	}

	conf := DefaultProxyConf(proxyType)
	if conf == nil {
		return nil, fmt.Errorf("invalid type [%s]", proxyType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, err
	}
	return conf, nil
}

// LocalSvrConf configures what location the client will to, or what
// plugin will be used.
type LocalSvrConf struct {
	// LocalIP specifies the IP address or host name to to.
	LocalIP string `ini:"local_ip" json:"local_ip"`
	// LocalPort specifies the port to to.
	LocalPort int `ini:"local_port" json:"local_port"`

	// Plugin specifies what plugin should be used for ng. If this value
	// is set, the LocalIp and LocalPort values will be ignored. By default,
	// this value is "".
	Plugin string `ini:"plugin" json:"plugin"`
	// PluginParams specify parameters to be passed to the plugin, if one is
	// being used. By default, this value is an empty map.
	PluginParams map[string]string `ini:"-"`
}

// HealthCheckConf configures health checking. This can be useful for load
// balancing purposes to detect and remove proxies to failing services.
type HealthCheckConf struct {
	// HealthCheckType specifies what protocol to use for health checking.
	// Valid values include "tcp", "http", and "". If this value is "", health
	// checking will not be performed. By default, this value is "".
	//
	// If the type is "tcp", a connection will be attempted to the target
	// server. If a connection cannot be established, the health check fails.
	//
	// If the type is "http", a GET request will be made to the endpoint
	// specified by HealthCheckURL. If the response is not a 200, the health
	// check fails.
	HealthCheckType string `ini:"health_check_type" json:"health_check_type"` // tcp | http
	// HealthCheckTimeoutS specifies the number of seconds to wait for a health
	// check attempt to connect. If the timeout is reached, this counts as a
	// health check failure. By default, this value is 3.
	HealthCheckTimeoutS int `ini:"health_check_timeout_s" json:"health_check_timeout_s"`
	// HealthCheckMaxFailed specifies the number of allowed failures before the
	// is stopped. By default, this value is 1.
	HealthCheckMaxFailed int `ini:"health_check_max_failed" json:"health_check_max_failed"`
	// HealthCheckIntervalS specifies the time in seconds between health
	// checks. By default, this value is 10.
	HealthCheckIntervalS int `ini:"health_check_interval_s" json:"health_check_interval_s"`
	// HealthCheckURL specifies the address to send health checks to if the
	// health check type is "http".
	HealthCheckURL string `ini:"health_check_url" json:"health_check_url"`
	// HealthCheckAddr specifies the address to connect to if the health check
	// type is "tcp".
	HealthCheckAddr string `ini:"-"`
}

// BaseProxyConf provides configuration info that is common to all types.
type BaseProxyConf struct {
	// ProxyName is the name of this
	ProxyName string `ini:"name" json:"name"`
	// ProxyType specifies the type of this  Valid values include "tcp",
	// "udp", "http", "https", "stcp", and "xtcp". By default, this value is
	// "tcp".
	ProxyType string `ini:"type" json:"type"`

	// UseEncryption controls whether or not communication with the server will
	// be encrypted. Encryption is done using the tokens supplied in the server
	// and client configuration. By default, this value is false.
	UseEncryption bool `ini:"use_encryption" json:"use_encryption"`
	// UseCompression controls whether or not communication with the server
	// will be compressed. By default, this value is false.
	UseCompression bool `ini:"use_compression" json:"use_compression"`
	// Group specifies which group the is a part of. The server will use
	// this information to load balance proxies in the same group. If the value
	// is "", this will not be in a group. By default, this value is "".
	Group string `ini:"group" json:"group"`
	// GroupKey specifies a group key, which should be the same among proxies
	// of the same group. By default, this value is "".
	GroupKey string `ini:"group_key" json:"group_key"`

	// ProxyProtocolVersion specifies which protocol version to use. Valid
	// values include "v1", "v2", and "". If the value is "", a protocol
	// version will be automatically selected. By default, this value is "".
	ProxyProtocolVersion string `ini:"proxy_protocol_version" json:"proxy_protocol_version"`

	// BandwidthLimit limit the bandwidth
	// 0 means no limit
	BandwidthLimit types.BandwidthQuantity `ini:"bandwidth_limit" json:"bandwidth_limit"`
	// BandwidthLimitMode specifies whether to limit the bandwidth on the
	// client or server side. Valid values include "client" and "server".
	// By default, this value is "client".
	BandwidthLimitMode string `ini:"bandwidth_limit_mode" json:"bandwidth_limit_mode"`

	// meta info for each proxy
	Metas map[string]string `ini:"-" json:"metas"`

	LocalSvrConf    `ini:",extends"`
	HealthCheckConf `ini:",extends"`
}

// Base
func (cfg *BaseProxyConf) GetBaseConfig() *BaseProxyConf {
	return cfg
}

// BaseProxyConf apply custom logic changes.
func (cfg *BaseProxyConf) decorate(_ string, name string, section *ini.Section) error {
	cfg.ProxyName = name
	// metas_xxx
	cfg.Metas = GetMapWithoutPrefix(section.KeysHash(), "meta_")

	// bandwidth_limit
	if bandwidth, err := section.GetKey("bandwidth_limit"); err == nil {
		cfg.BandwidthLimit, err = types.NewBandwidthQuantity(bandwidth.String())
		if err != nil {
			return err
		}
	}

	// plugin_xxx
	cfg.LocalSvrConf.PluginParams = GetMapByPrefix(section.KeysHash(), "plugin_")
	return nil
}

type DomainConf struct {
	CustomDomains []string `ini:"custom_domains" json:"custom_domains"`
	SubDomain     string   `ini:"subdomain" json:"subdomain"`
}

type RoleServerCommonConf struct {
	Role       string   `ini:"role" json:"role"`
	Sk         string   `ini:"sk" json:"sk"`
	AllowUsers []string `ini:"allow_users" json:"allow_users"`
}

// HTTP
type HTTPProxyConf struct {
	BaseProxyConf `ini:",extends"`
	DomainConf    `ini:",extends"`

	Locations         []string          `ini:"locations" json:"locations"`
	HTTPUser          string            `ini:"http_user" json:"http_user"`
	HTTPPwd           string            `ini:"http_pwd" json:"http_pwd"`
	HostHeaderRewrite string            `ini:"host_header_rewrite" json:"host_header_rewrite"`
	Headers           map[string]string `ini:"-" json:"headers"`
	RouteByHTTPUser   string            `ini:"route_by_http_user" json:"route_by_http_user"`
}

func (cfg *HTTPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	cfg.Headers = GetMapWithoutPrefix(section.KeysHash(), "header_")
	return nil
}

// HTTPS
type HTTPSProxyConf struct {
	BaseProxyConf `ini:",extends"`
	DomainConf    `ini:",extends"`
}

func (cfg *HTTPSProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	return nil
}

// TCP
type TCPProxyConf struct {
	BaseProxyConf `ini:",extends"`
	RemotePort    int `ini:"remote_port" json:"remote_port"`
}

func (cfg *TCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

// UDP
type UDPProxyConf struct {
	BaseProxyConf `ini:",extends"`

	RemotePort int `ini:"remote_port" json:"remote_port"`
}

func (cfg *UDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

// TCPMux
type TCPMuxProxyConf struct {
	BaseProxyConf   `ini:",extends"`
	DomainConf      `ini:",extends"`
	HTTPUser        string `ini:"http_user" json:"http_user,omitempty"`
	HTTPPwd         string `ini:"http_pwd" json:"http_pwd,omitempty"`
	RouteByHTTPUser string `ini:"route_by_http_user" json:"route_by_http_user"`

	Multiplexer string `ini:"multiplexer"`
}

func (cfg *TCPMuxProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

// STCP
type STCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *STCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	if cfg.Role == "" {
		cfg.Role = "server"
	}
	return nil
}

// XTCP
type XTCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *XTCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	if cfg.Role == "" {
		cfg.Role = "server"
	}
	return nil
}

// SUDP
type SUDPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

func (cfg *SUDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	return nil
}

func preUnmarshalFromIni(cfg ProxyConf, prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.GetBaseConfig().decorate(prefix, name, section)
	if err != nil {
		return err
	}

	return nil
}
