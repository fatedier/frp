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
func NewProxyConfFromMsg(pMsg *msg.NewProxy, svrconf ServerCommonConf) (ProxyConf, error) {
	if pMsg.ProxyType == "" {
		pMsg.ProxyType = consts.TCPProxy
	}

	conf := DefaultProxyConf(pMsg.ProxyType)
	if conf == nil {
		return nil, fmt.Errorf("proxy [%s] type [%s] error", pMsg.ProxyName, pMsg.ProxyType)
	}

	conf.UnmarshalFromMsg(pMsg)

	err := conf.CheckForSvr(svrconf)
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

func (c *BaseProxyConf) GetBaseInfo() *BaseProxyConf {
	return c
}

func (c *BaseProxyConf) compare(cmp *BaseProxyConf) bool {
	if c.ProxyName != cmp.ProxyName ||
		c.ProxyType != cmp.ProxyType ||
		c.UseEncryption != cmp.UseEncryption ||
		c.UseCompression != cmp.UseCompression ||
		c.Group != cmp.Group ||
		c.GroupKey != cmp.GroupKey ||
		c.ProxyProtocolVersion != cmp.ProxyProtocolVersion ||
		!c.BandwidthLimit.Equal(&cmp.BandwidthLimit) ||
		!reflect.DeepEqual(c.Metas, cmp.Metas) {
		return false
	}

	if !reflect.DeepEqual(c.LocalSvrConf, cmp.LocalSvrConf) {
		return false
	}
	if !reflect.DeepEqual(c.HealthCheckConf, cmp.HealthCheckConf) {
		return false
	}

	return true
}

// BaseProxyConf apply custom logic changes.
func (c *BaseProxyConf) decorate(prefix string, name string, section *ini.Section) error {
	// proxy_name
	c.ProxyName = prefix + name

	// metas_xxx
	c.Metas = GetMapWithoutPrefix(section.KeysHash(), "meta_")

	// bandwidth_limit
	if bandwidth, err := section.GetKey("bandwidth_limit"); err == nil {
		c.BandwidthLimit, err = NewBandwidthQuantity(bandwidth.String())
		if err != nil {
			return err
		}
	}

	// plugin_xxx
	c.LocalSvrConf.PluginParams = GetMapByPrefix(section.KeysHash(), "plugin_")

	// custom logic code
	if c.HealthCheckType == "tcp" && c.Plugin == "" {
		c.HealthCheckAddr = c.LocalIP + fmt.Sprintf(":%d", c.LocalPort)
	}

	if c.HealthCheckType == "http" && c.Plugin == "" && c.HealthCheckURL != "" {
		s := fmt.Sprintf("http://%s:%d", c.LocalIP, c.LocalPort)
		if !strings.HasPrefix(c.HealthCheckURL, "/") {
			s += "/"
		}
		c.HealthCheckURL = s + c.HealthCheckURL
	}

	return nil
}

func (c *BaseProxyConf) unmarshalFromMsg(pMsg *msg.NewProxy) {
	c.ProxyName = pMsg.ProxyName
	c.ProxyType = pMsg.ProxyType
	c.UseEncryption = pMsg.UseEncryption
	c.UseCompression = pMsg.UseCompression
	c.Group = pMsg.Group
	c.GroupKey = pMsg.GroupKey
	c.Metas = pMsg.Metas
}

func (c *BaseProxyConf) marshalToMsg(pMsg *msg.NewProxy) {
	pMsg.ProxyName = c.ProxyName
	pMsg.ProxyType = c.ProxyType
	pMsg.UseEncryption = c.UseEncryption
	pMsg.UseCompression = c.UseCompression
	pMsg.Group = c.Group
	pMsg.GroupKey = c.GroupKey
	pMsg.Metas = c.Metas
}

func (c *BaseProxyConf) checkForCli() error {
	if c.ProxyProtocolVersion != "" {
		if c.ProxyProtocolVersion != "v1" && c.ProxyProtocolVersion != "v2" {
			return fmt.Errorf("no support proxy protocol version: %s", c.ProxyProtocolVersion)
		}
	}

	if err := c.LocalSvrConf.validate(); err != nil {
		return err
	}
	if err := c.HealthCheckConf.validate(); err != nil {
		return err
	}
	return nil
}

func (c *BaseProxyConf) checkForSvr(conf ServerCommonConf) error {
	return nil
}

// LocalSvrConf
func (c *LocalSvrConf) validate() (err error) {
	if c.Plugin == "" {
		if c.LocalIP == "" {
			err = fmt.Errorf("local ip or plugin is required")
			return
		}
		if c.LocalPort <= 0 {
			err = fmt.Errorf("error local_port")
			return
		}
	}
	return
}

// HealthCheckConf
func (c *HealthCheckConf) validate() error {
	if c.HealthCheckType != "" && c.HealthCheckType != "tcp" && c.HealthCheckType != "http" {
		return fmt.Errorf("unsupport health check type")
	}
	if c.HealthCheckType != "" {
		if c.HealthCheckType == "http" && c.HealthCheckURL == "" {
			return fmt.Errorf("health_check_url is required for health check type 'http'")
		}
	}
	return nil
}

// DomainSpec
func (c *DomainSpec) validate() error {
	if len(c.CustomDomains) == 0 && c.SubDomain == "" {
		return fmt.Errorf("custom_domains and subdomain should set at least one of them")
	}
	return nil
}

func (c *DomainSpec) validateForCli() error {
	if err := c.validate(); err != nil {
		return err
	}
	return nil
}

func (c *DomainSpec) validateForSvr(svrconf ServerCommonConf) error {
	if err := c.validate(); err != nil {
		return err
	}

	for _, domain := range c.CustomDomains {
		if svrconf.SubDomainHost != "" && len(strings.Split(svrconf.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, svrconf.SubDomainHost) {
				return fmt.Errorf("custom domain [%s] should not belong to subdomain_host [%s]", domain, svrconf.SubDomainHost)
			}
		}
	}

	if c.SubDomain != "" {
		if svrconf.SubDomainHost == "" {
			return fmt.Errorf("subdomain is not supported because this feature is not enabled in remote frps")
		}
		if strings.Contains(c.SubDomain, ".") || strings.Contains(c.SubDomain, "*") {
			return fmt.Errorf("'.' and '*' is not supported in subdomain")
		}
	}
	return nil
}

// HTTP
var _ ProxyConf = &HTTPProxyConf{}

func (c *HTTPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*HTTPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.HTTPProxySpec, cmp.HTTPProxySpec) {
		return false
	}

	return true
}

func (c *HTTPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	c.Headers = GetMapWithoutPrefix(section.KeysHash(), "header_")

	return nil
}

func (c *HTTPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.CustomDomains = pMsg.CustomDomains
	c.SubDomain = pMsg.SubDomain
	c.Locations = pMsg.Locations
	c.HostHeaderRewrite = pMsg.HostHeaderRewrite
	c.HTTPUser = pMsg.HTTPUser
	c.HTTPPwd = pMsg.HTTPPwd
	c.Headers = pMsg.Headers
}

func (c *HTTPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = c.CustomDomains
	pMsg.SubDomain = c.SubDomain
	pMsg.Locations = c.Locations
	pMsg.HostHeaderRewrite = c.HostHeaderRewrite
	pMsg.HTTPUser = c.HTTPUser
	pMsg.HTTPPwd = c.HTTPPwd
	pMsg.Headers = c.Headers
}

func (c *HTTPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if err := c.DomainSpec.validateForCli(); err != nil {
		return err
	}

	return nil
}

func (c *HTTPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	if svrconf.VhostHTTPPort == 0 {
		return fmt.Errorf("type [http] not support when vhost_http_port is not set")
	}

	if err := c.DomainSpec.validateForSvr(svrconf); err != nil {
		return fmt.Errorf("proxy [%s] domain conf check error: %v", c.ProxyName, err)
	}

	return nil
}

// HTTPS
var _ ProxyConf = &HTTPSProxyConf{}

func (c *HTTPSProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*HTTPSProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.HTTPSProxySpec, cmp.HTTPSProxySpec) {
		return false
	}

	return true
}

func (c *HTTPSProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (c *HTTPSProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.CustomDomains = pMsg.CustomDomains
	c.SubDomain = pMsg.SubDomain
}

func (c *HTTPSProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = c.CustomDomains
	pMsg.SubDomain = c.SubDomain
}

func (c *HTTPSProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if err := c.DomainSpec.validateForCli(); err != nil {
		return err
	}

	return nil
}

func (c *HTTPSProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	if svrconf.VhostHTTPSPort == 0 {
		return fmt.Errorf("type [https] not support when vhost_https_port is not set")
	}

	if err := c.DomainSpec.validateForSvr(svrconf); err != nil {
		return fmt.Errorf("proxy [%s] domain conf check error: %v", c.ProxyName, err)
	}

	return nil
}

// TCP
var _ ProxyConf = &TCPProxyConf{}

func (c *TCPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*TCPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.TCPProxySpec, cmp.TCPProxySpec) {
		return false
	}

	return true
}

func (c *TCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (c *TCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.RemotePort = pMsg.RemotePort
}

func (c *TCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.RemotePort = c.RemotePort
}

func (c *TCPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists

	return nil
}

func (c *TCPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	return nil
}

// TCPMux
var _ ProxyConf = &TCPMuxProxyConf{}

func (c *TCPMuxProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*TCPMuxProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.TCPMuxProxySpec, cmp.TCPMuxProxySpec) {
		return false
	}

	return true
}

func (c *TCPMuxProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (c *TCPMuxProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.CustomDomains = pMsg.CustomDomains
	c.SubDomain = pMsg.SubDomain
	c.Multiplexer = pMsg.Multiplexer
}

func (c *TCPMuxProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.CustomDomains = c.CustomDomains
	pMsg.SubDomain = c.SubDomain
	pMsg.Multiplexer = c.Multiplexer
}

func (c *TCPMuxProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if err := c.DomainSpec.validateForCli(); err != nil {
		return err
	}

	if c.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("parse conf error: incorrect multiplexer [%s]", c.Multiplexer)
	}

	return nil
}

func (c *TCPMuxProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	if c.Multiplexer != consts.HTTPConnectTCPMultiplexer {
		return fmt.Errorf("proxy [%s] incorrect multiplexer [%s]", c.ProxyName, c.Multiplexer)
	}

	if c.Multiplexer == consts.HTTPConnectTCPMultiplexer && svrconf.TCPMuxHTTPConnectPort == 0 {
		return fmt.Errorf("proxy [%s] type [tcpmux] with multiplexer [httpconnect] requires tcpmux_httpconnect_port configuration", c.ProxyName)
	}

	if err := c.DomainSpec.validateForSvr(svrconf); err != nil {
		return fmt.Errorf("proxy [%s] domain conf check error: %v", c.ProxyName, err)
	}

	return nil
}

// STCP
var _ ProxyConf = &STCPProxyConf{}

func (c *STCPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*STCPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.STCPProxySpec, cmp.STCPProxySpec) {
		return false
	}

	return true
}

func (c *STCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	if c.Role == "" {
		c.Role = "server"
	}

	return nil
}

// Only for role server.
func (c *STCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.Sk = pMsg.Sk
}

func (c *STCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = c.Sk
}

func (c *STCPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if c.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return nil
}

func (c *STCPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	return nil
}

// XTCP
var _ ProxyConf = &XTCPProxyConf{}

func (c *XTCPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*XTCPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.XTCPProxySpec, cmp.XTCPProxySpec) {
		return false
	}

	return true
}

func (c *XTCPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists
	if c.Role == "" {
		c.Role = "server"
	}

	return nil
}

// Only for role server.
func (c *XTCPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.Sk = pMsg.Sk
}

func (c *XTCPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = c.Sk
}

func (c *XTCPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if c.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return nil
}

func (c *XTCPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	return nil
}

// UDP

var _ ProxyConf = &UDPProxyConf{}

func (c *UDPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*UDPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.UDPProxySpec, cmp.UDPProxySpec) {
		return false
	}

	return true
}

func (c *UDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

func (c *UDPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.RemotePort = pMsg.RemotePort
}

func (c *UDPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.RemotePort = c.RemotePort
}

func (c *UDPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists

	return nil
}

func (c *UDPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	return nil
}

// SUDP

var _ ProxyConf = &SUDPProxyConf{}

func (c *SUDPProxyConf) Compare(conf ProxyConf) bool {
	cmp, ok := conf.(*SUDPProxyConf)
	if !ok {
		return false
	}

	if !c.BaseProxyConf.compare(&cmp.BaseProxyConf) {
		return false
	}

	// Add custom logic equal if exists.
	if !reflect.DeepEqual(c.SUDPProxySpec, cmp.SUDPProxySpec) {
		return false
	}

	return true
}

func (c *SUDPProxyConf) UnmarshalFromIni(prefix string, name string, section *ini.Section) error {
	err := section.MapTo(c)
	if err != nil {
		return err
	}

	err = c.BaseProxyConf.decorate(prefix, name, section)
	if err != nil {
		return err
	}

	// Add custom logic unmarshal if exists

	return nil
}

// Only for role server.
func (c *SUDPProxyConf) UnmarshalFromMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.unmarshalFromMsg(pMsg)

	// Add custom logic unmarshal if exists
	c.Sk = pMsg.Sk
}

func (c *SUDPProxyConf) MarshalToMsg(pMsg *msg.NewProxy) {
	c.BaseProxyConf.marshalToMsg(pMsg)

	// Add custom logic marshal if exists
	pMsg.Sk = c.Sk
}

func (c *SUDPProxyConf) CheckForCli() error {
	if err := c.BaseProxyConf.checkForCli(); err != nil {
		return err
	}

	// Add custom logic check if exists
	if c.Role != "server" {
		return fmt.Errorf("role should be 'server'")
	}

	return nil
}

func (c *SUDPProxyConf) CheckForSvr(svrconf ServerCommonConf) error {
	return nil
}
