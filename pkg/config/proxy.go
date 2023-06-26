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
	"net"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/msg"
)

// Proxy
var (
	proxyConfTypeMap = map[string]reflect.Type{
		consts.TCPProxy:    reflect.TypeOf(TCPProxyConf{}),
		consts.TCPMuxProxy: reflect.TypeOf(TCPMuxProxyConf{}),
		consts.UDPProxy:    reflect.TypeOf(UDPProxyConf{}),
		consts.HTTPProxy:   reflect.TypeOf(HTTPProxyConf{}),
		consts.HTTPSProxy:  reflect.TypeOf(HTTPSProxyConf{}),
		consts.STCPProxy:   reflect.TypeOf(STCPProxyConf{}),
		consts.XTCPProxy:   reflect.TypeOf(XTCPProxyConf{}),
		consts.SUDPProxy:   reflect.TypeOf(SUDPProxyConf{}),
	}
)

func NewConfByType(proxyType string) ProxyConf {
	v, ok := proxyConfTypeMap[proxyType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(ProxyConf)
	return cfg
}

type ProxyConf interface {
	// GetBaseConfig returns the BaseProxyConf for this config.
	GetBaseConfig() *BaseProxyConf
	// SetDefaultValues sets the default values for this config.
	SetDefaultValues()
	// UnmarshalFromMsg unmarshals a msg.NewProxy message into this config.
	// This function will be called on the frps side.
	UnmarshalFromMsg(*msg.NewProxy)
	// UnmarshalFromIni unmarshals a ini.Section into this config. This function
	// will be called on the frpc side.
	UnmarshalFromIni(string, string, *ini.Section) error
	// MarshalToMsg marshals this config into a msg.NewProxy message. This
	// function will be called on the frpc side.
	MarshalToMsg(*msg.NewProxy)
	// ValidateForClient checks that the config is valid for the frpc side.
	ValidateForClient() error
	// ValidateForServer checks that the config is valid for the frps side.
	ValidateForServer(ServerCommonConf) error
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
	BandwidthLimit BandwidthQuantity `ini:"bandwidth_limit" json:"bandwidth_limit"`
	// BandwidthLimitMode specifies whether to limit the bandwidth on the
	// client or server side. Valid values include "client" and "server".
	// By default, this value is "client".
	BandwidthLimitMode string `ini:"bandwidth_limit_mode" json:"bandwidth_limit_mode"`

	// meta info for each proxy
	Metas map[string]string `ini:"-" json:"metas"`

	LocalSvrConf    `ini:",extends"`
	HealthCheckConf `ini:",extends"`
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

func (cfg *RoleServerCommonConf) setDefaultValues() {
	cfg.Role = "server"
}

func (cfg *RoleServerCommonConf) marshalToMsg(m *msg.NewProxy) {
	m.Sk = cfg.Sk
	m.AllowUsers = cfg.AllowUsers
}

func (cfg *RoleServerCommonConf) unmarshalFromMsg(m *msg.NewProxy) {
	cfg.Sk = m.Sk
	cfg.AllowUsers = m.AllowUsers
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

// HTTPS
type HTTPSProxyConf struct {
	BaseProxyConf `ini:",extends"`
	DomainConf    `ini:",extends"`
}

// TCP
type TCPProxyConf struct {
	BaseProxyConf `ini:",extends"`
	RemotePort    int `ini:"remote_port" json:"remote_port"`
}

// UDP
type UDPProxyConf struct {
	BaseProxyConf `ini:",extends"`

	RemotePort int `ini:"remote_port" json:"remote_port"`
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

// STCP
type STCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

// XTCP
type XTCPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

// SUDP
type SUDPProxyConf struct {
	BaseProxyConf        `ini:",extends"`
	RoleServerCommonConf `ini:",extends"`
}

// Proxy Conf Loader
// DefaultProxyConf creates a empty ProxyConf object by proxyType.
// If proxyType doesn't exist, return nil.
func DefaultProxyConf(proxyType string) ProxyConf {
	conf := NewConfByType(proxyType)
	if conf != nil {
		conf.SetDefaultValues()
	}
	return conf
}

// Proxy loaded from ini
func NewProxyConfFromIni(prefix, name string, section *ini.Section) (ProxyConf, error) {
	// section.Key: if key not exists, section will set it with default value.
	proxyType := section.Key("type").String()
	if proxyType == "" {
		proxyType = consts.TCPProxy
	}

	conf := DefaultProxyConf(proxyType)
	if conf == nil {
		return nil, fmt.Errorf("invalid type [%s]", proxyType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, err
	}

	if err := conf.ValidateForClient(); err != nil {
		return nil, err
	}
	return conf, nil
}

// Proxy loaded from msg
func NewProxyConfFromMsg(m *msg.NewProxy, serverCfg ServerCommonConf) (ProxyConf, error) {
	if m.ProxyType == "" {
		m.ProxyType = consts.TCPProxy
	}

	conf := DefaultProxyConf(m.ProxyType)
	if conf == nil {
		return nil, fmt.Errorf("proxy [%s] type [%s] error", m.ProxyName, m.ProxyType)
	}

	conf.UnmarshalFromMsg(m)

	err := conf.ValidateForServer(serverCfg)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

// Base
func (cfg *BaseProxyConf) GetBaseConfig() *BaseProxyConf {
	return cfg
}

func (cfg *BaseProxyConf) SetDefaultValues() {
	cfg.LocalSvrConf = LocalSvrConf{
		LocalIP: "127.0.0.1",
	}
	cfg.BandwidthLimitMode = BandwidthLimitModeClient
}

// BaseProxyConf apply custom logic changes.
func (cfg *BaseProxyConf) decorate(prefix string, name string, section *ini.Section) error {
	// proxy_name
	cfg.ProxyName = prefix + name

	// metas_xxx
	cfg.Metas = GetMapWithoutPrefix(section.KeysHash(), "meta_")

	// bandwidth_limit
	if bandwidth, err := section.GetKey("bandwidth_limit"); err == nil {
		cfg.BandwidthLimit, err = NewBandwidthQuantity(bandwidth.String())
		if err != nil {
			return err
		}
	}

	// plugin_xxx
	cfg.LocalSvrConf.PluginParams = GetMapByPrefix(section.KeysHash(), "plugin_")

	// custom logic code
	if cfg.HealthCheckType == "tcp" && cfg.Plugin == "" {
		cfg.HealthCheckAddr = cfg.LocalIP + fmt.Sprintf(":%d", cfg.LocalPort)
	}

	if cfg.HealthCheckType == "http" && cfg.Plugin == "" && cfg.HealthCheckURL != "" {
		s := "http://" + net.JoinHostPort(cfg.LocalIP, strconv.Itoa(cfg.LocalPort))
		if !strings.HasPrefix(cfg.HealthCheckURL, "/") {
			s += "/"
		}
		cfg.HealthCheckURL = s + cfg.HealthCheckURL
	}

	return nil
}

func (cfg *BaseProxyConf) marshalToMsg(m *msg.NewProxy) {
	m.ProxyName = cfg.ProxyName
	m.ProxyType = cfg.ProxyType
	m.UseEncryption = cfg.UseEncryption
	m.UseCompression = cfg.UseCompression
	m.BandwidthLimit = cfg.BandwidthLimit.String()
	// leave it empty for default value to reduce traffic
	if cfg.BandwidthLimitMode != "client" {
		m.BandwidthLimitMode = cfg.BandwidthLimitMode
	}
	m.Group = cfg.Group
	m.GroupKey = cfg.GroupKey
	m.Metas = cfg.Metas
}

func (cfg *BaseProxyConf) unmarshalFromMsg(m *msg.NewProxy) {
	cfg.ProxyName = m.ProxyName
	cfg.ProxyType = m.ProxyType
	cfg.UseEncryption = m.UseEncryption
	cfg.UseCompression = m.UseCompression
	if m.BandwidthLimit != "" {
		cfg.BandwidthLimit, _ = NewBandwidthQuantity(m.BandwidthLimit)
	}
	if m.BandwidthLimitMode != "" {
		cfg.BandwidthLimitMode = m.BandwidthLimitMode
	}
	cfg.Group = m.Group
	cfg.GroupKey = m.GroupKey
	cfg.Metas = m.Metas
}

func (cfg *BaseProxyConf) validateForClient() (err error) {
	if cfg.ProxyProtocolVersion != "" {
		if cfg.ProxyProtocolVersion != "v1" && cfg.ProxyProtocolVersion != "v2" {
			return fmt.Errorf("no support proxy protocol version: %s", cfg.ProxyProtocolVersion)
		}
	}

	if cfg.BandwidthLimitMode != "client" && cfg.BandwidthLimitMode != "server" {
		return fmt.Errorf("bandwidth_limit_mode should be client or server")
	}

	if err = cfg.LocalSvrConf.validateForClient(); err != nil {
		return
	}
	if err = cfg.HealthCheckConf.validateForClient(); err != nil {
		return
	}
	return nil
}

func (cfg *BaseProxyConf) validateForServer() (err error) {
	if cfg.BandwidthLimitMode != "client" && cfg.BandwidthLimitMode != "server" {
		return fmt.Errorf("bandwidth_limit_mode should be client or server")
	}
	return nil
}

// DomainConf
func (cfg *DomainConf) check() (err error) {
	if len(cfg.CustomDomains) == 0 && cfg.SubDomain == "" {
		err = fmt.Errorf("custom_domains and subdomain should set at least one of them")
		return
	}
	return
}

func (cfg *DomainConf) validateForClient() (err error) {
	if err = cfg.check(); err != nil {
		return
	}
	return
}

func (cfg *DomainConf) validateForServer(serverCfg ServerCommonConf) (err error) {
	if err = cfg.check(); err != nil {
		return
	}

	for _, domain := range cfg.CustomDomains {
		if serverCfg.SubDomainHost != "" && len(strings.Split(serverCfg.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, serverCfg.SubDomainHost) {
				return fmt.Errorf("custom domain [%s] should not belong to subdomain_host [%s]", domain, serverCfg.SubDomainHost)
			}
		}
	}

	if cfg.SubDomain != "" {
		if serverCfg.SubDomainHost == "" {
			return fmt.Errorf("subdomain is not supported because this feature is not enabled in remote frps")
		}
		if strings.Contains(cfg.SubDomain, ".") || strings.Contains(cfg.SubDomain, "*") {
			return fmt.Errorf("'.' and '*' is not supported in subdomain")
		}
	}
	return nil
}

// LocalSvrConf
func (cfg *LocalSvrConf) validateForClient() (err error) {
	if cfg.Plugin == "" {
		if cfg.LocalIP == "" {
			err = fmt.Errorf("local ip or plugin is required")
			return
		}
		if cfg.LocalPort <= 0 {
			err = fmt.Errorf("error local_port")
			return
		}
	}
	return
}

// HealthCheckConf
func (cfg *HealthCheckConf) validateForClient() error {
	if cfg.HealthCheckType != "" && cfg.HealthCheckType != "tcp" && cfg.HealthCheckType != "http" {
		return fmt.Errorf("unsupport health check type")
	}
	if cfg.HealthCheckType != "" {
		if cfg.HealthCheckType == "http" && cfg.HealthCheckURL == "" {
			return fmt.Errorf("health_check_url is required for health check type 'http'")
		}
	}
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

// TCP
func (cfg *TCPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.RemotePort = m.RemotePort
}

func (cfg *TCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *TCPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	m.RemotePort = cfg.RemotePort
}

func (cfg *TCPProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists

	return
}

func (cfg *TCPProxyConf) ValidateForServer(serverCfg ServerCommonConf) error {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}
	return nil
}

// TCPMux
func (cfg *TCPMuxProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *TCPMuxProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = m.CustomDomains
	cfg.SubDomain = m.SubDomain
	cfg.Multiplexer = m.Multiplexer
	cfg.HTTPUser = m.HTTPUser
	cfg.HTTPPwd = m.HTTPPwd
	cfg.RouteByHTTPUser = m.RouteByHTTPUser
}

func (cfg *TCPMuxProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	m.CustomDomains = cfg.CustomDomains
	m.SubDomain = cfg.SubDomain
	m.Multiplexer = cfg.Multiplexer
	m.HTTPUser = cfg.HTTPUser
	m.HTTPPwd = cfg.HTTPPwd
	m.RouteByHTTPUser = cfg.RouteByHTTPUser
}

func (cfg *TCPMuxProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.validateForClient(); err != nil {
		return
	}

	if cfg.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("parse conf error: incorrect multiplexer [%s]", cfg.Multiplexer)
	}

	return
}

func (cfg *TCPMuxProxyConf) ValidateForServer(serverCfg ServerCommonConf) (err error) {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}

	if cfg.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("proxy [%s] incorrect multiplexer [%s]", cfg.ProxyName, cfg.Multiplexer)
	}

	if cfg.Multiplexer == consts.HTTPConnectTCPMultiplexer && serverCfg.TCPMuxHTTPConnectPort == 0 {
		return fmt.Errorf("proxy [%s] type [tcpmux] with multiplexer [httpconnect] requires tcpmux_httpconnect_port configuration", cfg.ProxyName)
	}

	if err = cfg.DomainConf.validateForServer(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// UDP
func (cfg *UDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *UDPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.RemotePort = m.RemotePort
}

func (cfg *UDPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	m.RemotePort = cfg.RemotePort
}

func (cfg *UDPProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists

	return
}

func (cfg *UDPProxyConf) ValidateForServer(serverCfg ServerCommonConf) error {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}
	return nil
}

// HTTP
func (cfg *HTTPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	cfg.Headers = GetMapWithoutPrefix(section.KeysHash(), "header_")
	return nil
}

func (cfg *HTTPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = m.CustomDomains
	cfg.SubDomain = m.SubDomain
	cfg.Locations = m.Locations
	cfg.HostHeaderRewrite = m.HostHeaderRewrite
	cfg.HTTPUser = m.HTTPUser
	cfg.HTTPPwd = m.HTTPPwd
	cfg.Headers = m.Headers
	cfg.RouteByHTTPUser = m.RouteByHTTPUser
}

func (cfg *HTTPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	m.CustomDomains = cfg.CustomDomains
	m.SubDomain = cfg.SubDomain
	m.Locations = cfg.Locations
	m.HostHeaderRewrite = cfg.HostHeaderRewrite
	m.HTTPUser = cfg.HTTPUser
	m.HTTPPwd = cfg.HTTPPwd
	m.Headers = cfg.Headers
	m.RouteByHTTPUser = cfg.RouteByHTTPUser
}

func (cfg *HTTPProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.validateForClient(); err != nil {
		return
	}

	return
}

func (cfg *HTTPProxyConf) ValidateForServer(serverCfg ServerCommonConf) (err error) {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}

	if serverCfg.VhostHTTPPort == 0 {
		return fmt.Errorf("type [http] not support when vhost_http_port is not set")
	}

	if err = cfg.DomainConf.validateForServer(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// HTTPS
func (cfg *HTTPSProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	return nil
}

func (cfg *HTTPSProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = m.CustomDomains
	cfg.SubDomain = m.SubDomain
}

func (cfg *HTTPSProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	m.CustomDomains = cfg.CustomDomains
	m.SubDomain = cfg.SubDomain
}

func (cfg *HTTPSProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.validateForClient(); err != nil {
		return
	}
	return
}

func (cfg *HTTPSProxyConf) ValidateForServer(serverCfg ServerCommonConf) (err error) {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}

	if serverCfg.VhostHTTPSPort == 0 {
		return fmt.Errorf("type [https] not support when vhost_https_port is not set")
	}

	if err = cfg.DomainConf.validateForServer(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// SUDP
func (cfg *SUDPProxyConf) SetDefaultValues() {
	cfg.BaseProxyConf.SetDefaultValues()
	cfg.RoleServerCommonConf.setDefaultValues()
}

func (cfg *SUDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := preUnmarshalFromIni(cfg, prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	return nil
}

// Only for role server.
func (cfg *SUDPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.RoleServerCommonConf.unmarshalFromMsg(m)
}

func (cfg *SUDPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	cfg.RoleServerCommonConf.marshalToMsg(m)
}

func (cfg *SUDPProxyConf) ValidateForClient() (err error) {
	if err := cfg.BaseProxyConf.validateForClient(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return nil
}

func (cfg *SUDPProxyConf) ValidateForServer(serverCfg ServerCommonConf) error {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}
	return nil
}

// STCP
func (cfg *STCPProxyConf) SetDefaultValues() {
	cfg.BaseProxyConf.SetDefaultValues()
	cfg.RoleServerCommonConf.setDefaultValues()
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

// Only for role server.
func (cfg *STCPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.RoleServerCommonConf.unmarshalFromMsg(m)
}

func (cfg *STCPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	cfg.RoleServerCommonConf.marshalToMsg(m)
}

func (cfg *STCPProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return
}

func (cfg *STCPProxyConf) ValidateForServer(serverCfg ServerCommonConf) error {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}
	return nil
}

// XTCP
func (cfg *XTCPProxyConf) SetDefaultValues() {
	cfg.BaseProxyConf.SetDefaultValues()
	cfg.RoleServerCommonConf.setDefaultValues()
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

// Only for role server.
func (cfg *XTCPProxyConf) UnmarshalFromMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(m)

	// Add custom logic unmarshal if exists
	cfg.RoleServerCommonConf.unmarshalFromMsg(m)
}

func (cfg *XTCPProxyConf) MarshalToMsg(m *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(m)

	// Add custom logic marshal if exists
	cfg.RoleServerCommonConf.marshalToMsg(m)
}

func (cfg *XTCPProxyConf) ValidateForClient() (err error) {
	if err = cfg.BaseProxyConf.validateForClient(); err != nil {
		return
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}
	return
}

func (cfg *XTCPProxyConf) ValidateForServer(serverCfg ServerCommonConf) error {
	if err := cfg.BaseProxyConf.validateForServer(); err != nil {
		return err
	}
	return nil
}
