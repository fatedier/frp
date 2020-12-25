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
	"reflect"
	"strings"

	"github.com/fatedier/frp/pkg/consts"
	"github.com/fatedier/frp/pkg/msg"

	"gopkg.in/ini.v1"
)

// Proxy Conf Loader
// DefaultProxyConf creates a empty ProxyConf object by proxyType.
// If proxyType doesn't exist, return nil.
func DefaultProxyConf(proxyType string) ProxyConf {
	var conf ProxyConf
	switch proxyType {
	case consts.TCPProxy:
		conf = &TCPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
		}
	case consts.TCPMuxProxy:
		conf = &TCPMuxProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
		}
	case consts.UDPProxy:
		conf = &UDPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
		}
	case consts.HTTPProxy:
		conf = &HTTPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
		}
	case consts.HTTPSProxy:
		conf = &HTTPSProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
		}
	case consts.STCPProxy:
		conf = &STCPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
			STCPProxySpec: STCPProxySpec{
				Role: "server",
			},
		}
	case consts.XTCPProxy:
		conf = &XTCPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
			XTCPProxySpec: XTCPProxySpec{
				Role: "server",
			},
		}
	case consts.SUDPProxy:
		conf = &SUDPProxyConf{
			BaseProxyConf: defaultBaseProxyConf(proxyType),
			SUDPProxySpec: SUDPProxySpec{
				Role: "server",
			},
		}
	default:
		return nil
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
		return nil, fmt.Errorf("proxy [%s] type [%s] error", name, proxyType)
	}

	if err := conf.UnmarshalFromIni(prefix, name, section); err != nil {
		return nil, err
	}

	if err := conf.CheckForCli(); err != nil {
		return nil, err
	}

	return conf, nil
}

// Proxy loaded from msg
func NewProxyConfFromMsg(pMsg *msg.NewProxy, serverCfg ServerCommonConf) (ProxyConf, error) {
	if pMsg.ProxyType == "" {
		pMsg.ProxyType = consts.TCPProxy
	}

	conf := DefaultProxyConf(pMsg.ProxyType)
	if conf == nil {
		return nil, fmt.Errorf("proxy [%s] type [%s] error", pMsg.ProxyName, pMsg.ProxyType)
	}

	conf.UnmarshalFromMsg(pMsg)

	err := conf.CheckForSvr(serverCfg)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

// Base
func defaultBaseProxyConf(proxyType string) BaseProxyConf {
	return BaseProxyConf{
		ProxyType: proxyType,
		LocalSvrConf: LocalSvrConf{
			LocalIP: "127.0.0.1",
		},
	}
}

func (cfg *BaseProxyConf) GetBaseInfo() *BaseProxyConf {
	return cfg
}

func (cfg *BaseProxyConf) compare(cmp *BaseProxyConf) bool {
	if cfg.ProxyName != cmp.ProxyName ||
		cfg.ProxyType != cmp.ProxyType ||
		cfg.UseEncryption != cmp.UseEncryption ||
		cfg.UseCompression != cmp.UseCompression ||
		cfg.Group != cmp.Group ||
		cfg.GroupKey != cmp.GroupKey ||
		cfg.ProxyProtocolVersion != cmp.ProxyProtocolVersion ||
		!cfg.BandwidthLimit.Equal(&cmp.BandwidthLimit) ||
		!reflect.DeepEqual(cfg.Metas, cmp.Metas) {
		return false
	}

	if !reflect.DeepEqual(cfg.LocalSvrConf, cmp.LocalSvrConf) {
		return false
	}
	if !reflect.DeepEqual(cfg.HealthCheckConf, cmp.HealthCheckConf) {
		return false
	}

	return true
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
		s := fmt.Sprintf("http://%s:%d", cfg.LocalIP, cfg.LocalPort)
		if !strings.HasPrefix(cfg.HealthCheckURL, "/") {
			s += "/"
		}
		cfg.HealthCheckURL = s + cfg.HealthCheckURL
	}

	return nil
}

func (cfg *BaseProxyConf) marshalToMsg(pMsg *msg.NewProxy) {
	pMsg.ProxyName = cfg.ProxyName
	pMsg.ProxyType = cfg.ProxyType
	pMsg.UseEncryption = cfg.UseEncryption
	pMsg.UseCompression = cfg.UseCompression
	pMsg.Group = cfg.Group
	pMsg.GroupKey = cfg.GroupKey
	pMsg.Metas = cfg.Metas
}

func (cfg *BaseProxyConf) unmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.ProxyName = pMsg.ProxyName
	cfg.ProxyType = pMsg.ProxyType
	cfg.UseEncryption = pMsg.UseEncryption
	cfg.UseCompression = pMsg.UseCompression
	cfg.Group = pMsg.Group
	cfg.GroupKey = pMsg.GroupKey
	cfg.Metas = pMsg.Metas
}

func (cfg *BaseProxyConf) checkForCli() (err error) {
	if cfg.ProxyProtocolVersion != "" {
		if cfg.ProxyProtocolVersion != "v1" && cfg.ProxyProtocolVersion != "v2" {
			return fmt.Errorf("no support proxy protocol version: %s", cfg.ProxyProtocolVersion)
		}
	}

	if err = cfg.LocalSvrConf.checkForCli(); err != nil {
		return
	}
	if err = cfg.HealthCheckConf.checkForCli(); err != nil {
		return
	}
	return nil
}

func (cfg *BaseProxyConf) checkForSvr(conf ServerCommonConf) error {
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

func (cfg *DomainConf) checkForCli() (err error) {
	if err = cfg.check(); err != nil {
		return
	}
	return
}

func (cfg *DomainConf) checkForSvr(serverCfg ServerCommonConf) (err error) {
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
func (cfg *LocalSvrConf) checkForCli() (err error) {
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
func (cfg *HealthCheckConf) checkForCli() error {
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

// TCP
var _ ProxyConf = &TCPProxyConf{}

func (cfg *TCPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*TCPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.TCPProxySpec, cmpConf.TCPProxySpec) {
		return false
	}

	return true
}

func (cfg *TCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.RemotePort = pMsg.RemotePort
}

func (cfg *TCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *TCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.RemotePort = cfg.RemotePort
}

func (cfg *TCPProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists

	return
}

func (cfg *TCPProxyConf) CheckForSvr(serverCfg ServerCommonConf) error {
	return nil
}

// TCPMux
var _ ProxyConf = &TCPMuxProxyConf{}

func (cfg *TCPMuxProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*TCPMuxProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.TCPMuxProxySpec, cmpConf.TCPMuxProxySpec) {
		return false
	}

	return true
}

func (cfg *TCPMuxProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *TCPMuxProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = pMsg.CustomDomains
	cfg.SubDomain = pMsg.SubDomain
	cfg.Multiplexer = pMsg.Multiplexer
}

func (cfg *TCPMuxProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = cfg.CustomDomains
	pMsg.SubDomain = cfg.SubDomain
	pMsg.Multiplexer = cfg.Multiplexer
}

func (cfg *TCPMuxProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.checkForCli(); err != nil {
		return
	}

	if cfg.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("parse conf error: incorrect multiplexer [%s]", cfg.Multiplexer)
	}

	return
}

func (cfg *TCPMuxProxyConf) CheckForSvr(serverCfg ServerCommonConf) (err error) {
	if cfg.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("proxy [%s] incorrect multiplexer [%s]", cfg.ProxyName, cfg.Multiplexer)
	}

	if cfg.Multiplexer == consts.HTTPConnectTCPMultiplexer && serverCfg.TCPMuxHTTPConnectPort == 0 {
		return fmt.Errorf("proxy [%s] type [tcpmux] with multiplexer [httpconnect] requires tcpmux_httpconnect_port configuration", cfg.ProxyName)
	}

	if err = cfg.DomainConf.checkForSvr(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// UDP
var _ ProxyConf = &UDPProxyConf{}

func (cfg *UDPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*UDPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.UDPProxySpec, cmpConf.UDPProxySpec) {
		return false
	}

	return true
}

func (cfg *UDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *UDPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.RemotePort = pMsg.RemotePort
}

func (cfg *UDPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.RemotePort = cfg.RemotePort
}

func (cfg *UDPProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists

	return
}

func (cfg *UDPProxyConf) CheckForSvr(serverCfg ServerCommonConf) error {
	return nil
}

// HTTP
var _ ProxyConf = &HTTPProxyConf{}

func (cfg *HTTPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*HTTPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.HTTPProxySpec, cmpConf.HTTPProxySpec) {
		return false
	}

	return true
}

func (cfg *HTTPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	cfg.Headers = GetMapWithoutPrefix(section.KeysHash(), "header_")

	return nil
}

func (cfg *HTTPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = pMsg.CustomDomains
	cfg.SubDomain = pMsg.SubDomain
	cfg.Locations = pMsg.Locations
	cfg.HostHeaderRewrite = pMsg.HostHeaderRewrite
	cfg.HTTPUser = pMsg.HTTPUser
	cfg.HTTPPwd = pMsg.HTTPPwd
	cfg.Headers = pMsg.Headers
}

func (cfg *HTTPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = cfg.CustomDomains
	pMsg.SubDomain = cfg.SubDomain
	pMsg.Locations = cfg.Locations
	pMsg.HostHeaderRewrite = cfg.HostHeaderRewrite
	pMsg.HTTPUser = cfg.HTTPUser
	pMsg.HTTPPwd = cfg.HTTPPwd
	pMsg.Headers = cfg.Headers
}

func (cfg *HTTPProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.checkForCli(); err != nil {
		return
	}

	return
}

func (cfg *HTTPProxyConf) CheckForSvr(serverCfg ServerCommonConf) (err error) {
	if serverCfg.VhostHTTPPort == 0 {
		return fmt.Errorf("type [http] not support when vhost_http_port is not set")
	}

	if err = cfg.DomainConf.checkForSvr(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// HTTPS
var _ ProxyConf = &HTTPSProxyConf{}

func (cfg *HTTPSProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*HTTPSProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.HTTPSProxySpec, cmpConf.HTTPSProxySpec) {
		return false
	}

	return true
}

func (cfg *HTTPSProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (cfg *HTTPSProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.CustomDomains = pMsg.CustomDomains
	cfg.SubDomain = pMsg.SubDomain
}

func (cfg *HTTPSProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = cfg.CustomDomains
	pMsg.SubDomain = cfg.SubDomain
}

func (cfg *HTTPSProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists
	if err = cfg.DomainConf.checkForCli(); err != nil {
		return
	}

	return
}

func (cfg *HTTPSProxyConf) CheckForSvr(serverCfg ServerCommonConf) (err error) {
	if serverCfg.VhostHTTPSPort == 0 {
		return fmt.Errorf("type [https] not support when vhost_https_port is not set")
	}

	if err = cfg.DomainConf.checkForSvr(serverCfg); err != nil {
		err = fmt.Errorf("proxy [%s] domain conf check error: %v", cfg.ProxyName, err)
		return
	}

	return
}

// SUDP
var _ ProxyConf = &SUDPProxyConf{}

func (cfg *SUDPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*SUDPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.SUDPProxySpec, cmpConf.SUDPProxySpec) {
		return false
	}

	return true
}

func (cfg *SUDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

// Only for role server.
func (cfg *SUDPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.Sk = pMsg.Sk
}

func (cfg *SUDPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = cfg.Sk
}

func (cfg *SUDPProxyConf) CheckForCli() (err error) {
	if err := cfg.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return nil
}

func (cfg *SUDPProxyConf) CheckForSvr(serverCfg ServerCommonConf) error {
	return nil
}

// STCP
var _ ProxyConf = &STCPProxyConf{}

func (cfg *STCPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*STCPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.STCPProxySpec, cmpConf.STCPProxySpec) {
		return false
	}

	return true
}

func (cfg *STCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
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
func (cfg *STCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.Sk = pMsg.Sk
}

func (cfg *STCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = cfg.Sk
}

func (cfg *STCPProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return
}

func (cfg *STCPProxyConf) CheckForSvr(serverCfg ServerCommonConf) error {
	return nil
}

// XTCP
var _ ProxyConf = &XTCPProxyConf{}

func (cfg *XTCPProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*XTCPProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(cfg.XTCPProxySpec, cmpConf.XTCPProxySpec) {
		return false
	}

	return true
}

func (cfg *XTCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(cfg)
	if err != nil {
		return err
	}

	err = cfg.BaseProxyConf.decorate(prefix, name, section)
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
func (cfg *XTCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	cfg.Sk = pMsg.Sk
}

func (cfg *XTCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = cfg.Sk
}

func (cfg *XTCPProxyConf) CheckForCli() (err error) {
	if err = cfg.BaseProxyConf.checkForCli(); err != nil {
		return
	}

	// Add custom logic check if exists
	if cfg.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return
}

func (cfg *XTCPProxyConf) CheckForSvr(serverCfg ServerCommonConf) error {
	return nil
}
