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
	"strconv"
	"strings"

	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/models/msg"

	ini "github.com/vaughan0/go-ini"
)

var proxyConfTypeMap map[string]reflect.Type

func init() {
	proxyConfTypeMap = make(map[string]reflect.Type)
	proxyConfTypeMap[consts.TcpProxy] = reflect.TypeOf(TcpProxyConf{})
	proxyConfTypeMap[consts.UdpProxy] = reflect.TypeOf(UdpProxyConf{})
	proxyConfTypeMap[consts.HttpProxy] = reflect.TypeOf(HttpProxyConf{})
	proxyConfTypeMap[consts.HttpsProxy] = reflect.TypeOf(HttpsProxyConf{})
	proxyConfTypeMap[consts.StcpProxy] = reflect.TypeOf(StcpProxyConf{})
	proxyConfTypeMap[consts.XtcpProxy] = reflect.TypeOf(XtcpProxyConf{})
}

// NewConfByType creates a empty ProxyConf object by proxyType.
// If proxyType isn't exist, return nil.
func NewConfByType(proxyType string) ProxyConf {
	v, ok := proxyConfTypeMap[proxyType]
	if !ok {
		return nil
	}
	cfg := reflect.New(v).Interface().(ProxyConf)
	return cfg
}

type ProxyConf interface {
	GetName() string
	GetType() string
	GetBaseInfo() *BaseProxyConf
	LoadFromMsg(pMsg *msg.NewProxy)
	LoadFromFile(name string, conf ini.Section) error
	UnMarshalToMsg(pMsg *msg.NewProxy)
	Check() error
	Compare(conf ProxyConf) bool
}

func NewProxyConf(pMsg *msg.NewProxy) (cfg ProxyConf, err error) {
	if pMsg.ProxyType == "" {
		pMsg.ProxyType = consts.TcpProxy
	}

	cfg = NewConfByType(pMsg.ProxyType)
	if cfg == nil {
		err = fmt.Errorf("proxy [%s] type [%s] error", pMsg.ProxyName, pMsg.ProxyType)
		return
	}
	cfg.LoadFromMsg(pMsg)
	err = cfg.Check()
	return
}

func NewProxyConfFromFile(name string, section ini.Section) (cfg ProxyConf, err error) {
	proxyType := section["type"]
	if proxyType == "" {
		proxyType = consts.TcpProxy
		section["type"] = consts.TcpProxy
	}
	cfg = NewConfByType(proxyType)
	if cfg == nil {
		err = fmt.Errorf("proxy [%s] type [%s] error", name, proxyType)
		return
	}
	err = cfg.LoadFromFile(name, section)
	return
}

// BaseProxy info
type BaseProxyConf struct {
	ProxyName string `json:"proxy_name"`
	ProxyType string `json:"proxy_type"`

	UseEncryption  bool `json:"use_encryption"`
	UseCompression bool `json:"use_compression"`
}

func (cfg *BaseProxyConf) GetName() string {
	return cfg.ProxyName
}

func (cfg *BaseProxyConf) GetType() string {
	return cfg.ProxyType
}

func (cfg *BaseProxyConf) GetBaseInfo() *BaseProxyConf {
	return cfg
}

func (cfg *BaseProxyConf) compare(cmp *BaseProxyConf) bool {
	if cfg.ProxyName != cmp.ProxyName ||
		cfg.ProxyType != cmp.ProxyType ||
		cfg.UseEncryption != cmp.UseEncryption ||
		cfg.UseCompression != cmp.UseCompression {
		return false
	}
	return true
}

func (cfg *BaseProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.ProxyName = pMsg.ProxyName
	cfg.ProxyType = pMsg.ProxyType
	cfg.UseEncryption = pMsg.UseEncryption
	cfg.UseCompression = pMsg.UseCompression
}

func (cfg *BaseProxyConf) LoadFromFile(name string, section ini.Section) error {
	var (
		tmpStr string
		ok     bool
	)
	if ClientCommonCfg.User != "" {
		cfg.ProxyName = ClientCommonCfg.User + "." + name
	} else {
		cfg.ProxyName = name
	}
	cfg.ProxyType = section["type"]

	tmpStr, ok = section["use_encryption"]
	if ok && tmpStr == "true" {
		cfg.UseEncryption = true
	}

	tmpStr, ok = section["use_compression"]
	if ok && tmpStr == "true" {
		cfg.UseCompression = true
	}
	return nil
}

func (cfg *BaseProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.ProxyName = cfg.ProxyName
	pMsg.ProxyType = cfg.ProxyType
	pMsg.UseEncryption = cfg.UseEncryption
	pMsg.UseCompression = cfg.UseCompression
}

// Bind info
type BindInfoConf struct {
	BindAddr   string `json:"bind_addr"`
	RemotePort int    `json:"remote_port"`
}

func (cfg *BindInfoConf) compare(cmp *BindInfoConf) bool {
	if cfg.BindAddr != cmp.BindAddr ||
		cfg.RemotePort != cmp.RemotePort {
		return false
	}
	return true
}

func (cfg *BindInfoConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BindAddr = ServerCommonCfg.ProxyBindAddr
	cfg.RemotePort = pMsg.RemotePort
}

func (cfg *BindInfoConf) LoadFromFile(name string, section ini.Section) (err error) {
	var (
		tmpStr string
		ok     bool
		v      int64
	)
	if tmpStr, ok = section["remote_port"]; ok {
		if v, err = strconv.ParseInt(tmpStr, 10, 64); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] remote_port error", name)
		} else {
			cfg.RemotePort = int(v)
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] remote_port not found", name)
	}
	return nil
}

func (cfg *BindInfoConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.RemotePort = cfg.RemotePort
}

func (cfg *BindInfoConf) check() (err error) {
	return nil
}

// Domain info
type DomainConf struct {
	CustomDomains []string `json:"custom_domains"`
	SubDomain     string   `json:"sub_domain"`
}

func (cfg *DomainConf) compare(cmp *DomainConf) bool {
	if strings.Join(cfg.CustomDomains, " ") != strings.Join(cmp.CustomDomains, " ") ||
		cfg.SubDomain != cmp.SubDomain {
		return false
	}
	return true
}

func (cfg *DomainConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.CustomDomains = pMsg.CustomDomains
	cfg.SubDomain = pMsg.SubDomain
}

func (cfg *DomainConf) LoadFromFile(name string, section ini.Section) (err error) {
	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["custom_domains"]; ok {
		cfg.CustomDomains = strings.Split(tmpStr, ",")
		for i, domain := range cfg.CustomDomains {
			cfg.CustomDomains[i] = strings.ToLower(strings.TrimSpace(domain))
		}
	}

	if tmpStr, ok = section["subdomain"]; ok {
		cfg.SubDomain = tmpStr
	}

	if len(cfg.CustomDomains) == 0 && cfg.SubDomain == "" {
		return fmt.Errorf("Parse conf error: proxy [%s] custom_domains and subdomain should set at least one of them", name)
	}
	return
}

func (cfg *DomainConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	pMsg.CustomDomains = cfg.CustomDomains
	pMsg.SubDomain = cfg.SubDomain
}

func (cfg *DomainConf) check() (err error) {
	for _, domain := range cfg.CustomDomains {
		if ServerCommonCfg.SubDomainHost != "" && len(strings.Split(ServerCommonCfg.SubDomainHost, ".")) < len(strings.Split(domain, ".")) {
			if strings.Contains(domain, ServerCommonCfg.SubDomainHost) {
				return fmt.Errorf("custom domain [%s] should not belong to subdomain_host [%s]", domain, ServerCommonCfg.SubDomainHost)
			}
		}
	}

	if cfg.SubDomain != "" {
		if ServerCommonCfg.SubDomainHost == "" {
			return fmt.Errorf("subdomain is not supported because this feature is not enabled by frps")
		}
		if strings.Contains(cfg.SubDomain, ".") || strings.Contains(cfg.SubDomain, "*") {
			return fmt.Errorf("'.' and '*' is not supported in subdomain")
		}
	}
	return nil
}

// Local service info
type LocalSvrConf struct {
	LocalIp   string `json:"-"`
	LocalPort int    `json:"-"`
}

func (cfg *LocalSvrConf) compare(cmp *LocalSvrConf) bool {
	if cfg.LocalIp != cmp.LocalIp ||
		cfg.LocalPort != cmp.LocalPort {
		return false
	}
	return true
}

func (cfg *LocalSvrConf) LoadFromFile(name string, section ini.Section) (err error) {
	if cfg.LocalIp = section["local_ip"]; cfg.LocalIp == "" {
		cfg.LocalIp = "127.0.0.1"
	}

	if tmpStr, ok := section["local_port"]; ok {
		if cfg.LocalPort, err = strconv.Atoi(tmpStr); err != nil {
			return fmt.Errorf("Parse conf error: proxy [%s] local_port error", name)
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] local_port not found", name)
	}
	return nil
}

type PluginConf struct {
	Plugin       string            `json:"-"`
	PluginParams map[string]string `json:"-"`
}

func (cfg *PluginConf) compare(cmp *PluginConf) bool {
	if cfg.Plugin != cmp.Plugin ||
		len(cfg.PluginParams) != len(cmp.PluginParams) {
		return false
	}
	for k, v := range cfg.PluginParams {
		value, ok := cmp.PluginParams[k]
		if !ok || v != value {
			return false
		}
	}
	return true
}

func (cfg *PluginConf) LoadFromFile(name string, section ini.Section) (err error) {
	cfg.Plugin = section["plugin"]
	cfg.PluginParams = make(map[string]string)
	if cfg.Plugin != "" {
		// get params begin with "plugin_"
		for k, v := range section {
			if strings.HasPrefix(k, "plugin_") {
				cfg.PluginParams[k] = v
			}
		}
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] no plugin info found", name)
	}
	return
}

// TCP
type TcpProxyConf struct {
	BaseProxyConf
	BindInfoConf

	LocalSvrConf
	PluginConf
}

func (cfg *TcpProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*TcpProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.BindInfoConf.compare(&cmpConf.BindInfoConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) ||
		!cfg.PluginConf.compare(&cmpConf.PluginConf) {
		return false
	}
	return true
}

func (cfg *TcpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.BindInfoConf.LoadFromMsg(pMsg)
}

func (cfg *TcpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.BindInfoConf.LoadFromFile(name, section); err != nil {
		return
	}

	if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
		if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
			return
		}
	}
	return
}

func (cfg *TcpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.BindInfoConf.UnMarshalToMsg(pMsg)
}

func (cfg *TcpProxyConf) Check() (err error) {
	err = cfg.BindInfoConf.check()
	return
}

// UDP
type UdpProxyConf struct {
	BaseProxyConf
	BindInfoConf

	LocalSvrConf
}

func (cfg *UdpProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*UdpProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.BindInfoConf.compare(&cmpConf.BindInfoConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) {
		return false
	}
	return true
}

func (cfg *UdpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.BindInfoConf.LoadFromMsg(pMsg)
}

func (cfg *UdpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.BindInfoConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
		return
	}
	return
}

func (cfg *UdpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.BindInfoConf.UnMarshalToMsg(pMsg)
}

func (cfg *UdpProxyConf) Check() (err error) {
	err = cfg.BindInfoConf.check()
	return
}

// HTTP
type HttpProxyConf struct {
	BaseProxyConf
	DomainConf

	LocalSvrConf
	PluginConf

	Locations         []string `json:"locations"`
	HostHeaderRewrite string   `json:"host_header_rewrite"`
	HttpUser          string   `json:"-"`
	HttpPwd           string   `json:"-"`
}

func (cfg *HttpProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*HttpProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.DomainConf.compare(&cmpConf.DomainConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) ||
		!cfg.PluginConf.compare(&cmpConf.PluginConf) ||
		strings.Join(cfg.Locations, " ") != strings.Join(cmpConf.Locations, " ") ||
		cfg.HostHeaderRewrite != cmpConf.HostHeaderRewrite ||
		cfg.HttpUser != cmpConf.HttpUser ||
		cfg.HttpPwd != cmpConf.HttpPwd {
		return false
	}
	return true
}

func (cfg *HttpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.DomainConf.LoadFromMsg(pMsg)

	cfg.Locations = pMsg.Locations
	cfg.HostHeaderRewrite = pMsg.HostHeaderRewrite
	cfg.HttpUser = pMsg.HttpUser
	cfg.HttpPwd = pMsg.HttpPwd
}

func (cfg *HttpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.DomainConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
		if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
			return
		}
	}

	var (
		tmpStr string
		ok     bool
	)
	if tmpStr, ok = section["locations"]; ok {
		cfg.Locations = strings.Split(tmpStr, ",")
	} else {
		cfg.Locations = []string{""}
	}

	cfg.HostHeaderRewrite = section["host_header_rewrite"]
	cfg.HttpUser = section["http_user"]
	cfg.HttpPwd = section["http_pwd"]
	return
}

func (cfg *HttpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.DomainConf.UnMarshalToMsg(pMsg)

	pMsg.Locations = cfg.Locations
	pMsg.HostHeaderRewrite = cfg.HostHeaderRewrite
	pMsg.HttpUser = cfg.HttpUser
	pMsg.HttpPwd = cfg.HttpPwd
}

func (cfg *HttpProxyConf) Check() (err error) {
	if ServerCommonCfg.VhostHttpPort == 0 {
		return fmt.Errorf("type [http] not support when vhost_http_port is not set")
	}
	err = cfg.DomainConf.check()
	return
}

// HTTPS
type HttpsProxyConf struct {
	BaseProxyConf
	DomainConf

	LocalSvrConf
	PluginConf
}

func (cfg *HttpsProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*HttpsProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.DomainConf.compare(&cmpConf.DomainConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) ||
		!cfg.PluginConf.compare(&cmpConf.PluginConf) {
		return false
	}
	return true
}

func (cfg *HttpsProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.DomainConf.LoadFromMsg(pMsg)
}

func (cfg *HttpsProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.DomainConf.LoadFromFile(name, section); err != nil {
		return
	}
	if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
		if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
			return
		}
	}
	return
}

func (cfg *HttpsProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	cfg.DomainConf.UnMarshalToMsg(pMsg)
}

func (cfg *HttpsProxyConf) Check() (err error) {
	if ServerCommonCfg.VhostHttpsPort == 0 {
		return fmt.Errorf("type [https] not support when vhost_https_port is not set")
	}
	err = cfg.DomainConf.check()
	return
}

// STCP
type StcpProxyConf struct {
	BaseProxyConf

	Role string `json:"role"`
	Sk   string `json:"sk"`

	// used in role server
	LocalSvrConf
	PluginConf

	// used in role visitor
	ServerName string `json:"server_name"`
	BindAddr   string `json:"bind_addr"`
	BindPort   int    `json:"bind_port"`
}

func (cfg *StcpProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*StcpProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) ||
		!cfg.PluginConf.compare(&cmpConf.PluginConf) ||
		cfg.Role != cmpConf.Role ||
		cfg.Sk != cmpConf.Sk ||
		cfg.ServerName != cmpConf.ServerName ||
		cfg.BindAddr != cmpConf.BindAddr ||
		cfg.BindPort != cmpConf.BindPort {
		return false
	}
	return true
}

// Only for role server.
func (cfg *StcpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.Sk = pMsg.Sk
}

func (cfg *StcpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}

	tmpStr := section["role"]
	if tmpStr == "" {
		tmpStr = "server"
	}
	if tmpStr == "server" || tmpStr == "visitor" {
		cfg.Role = tmpStr
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] incorrect role [%s]", name, tmpStr)
	}

	cfg.Sk = section["sk"]

	if tmpStr == "visitor" {
		prefix := section["prefix"]
		cfg.ServerName = prefix + section["server_name"]
		if cfg.BindAddr = section["bind_addr"]; cfg.BindAddr == "" {
			cfg.BindAddr = "127.0.0.1"
		}

		if tmpStr, ok := section["bind_port"]; ok {
			if cfg.BindPort, err = strconv.Atoi(tmpStr); err != nil {
				return fmt.Errorf("Parse conf error: proxy [%s] bind_port error", name)
			}
		} else {
			return fmt.Errorf("Parse conf error: proxy [%s] bind_port not found", name)
		}
	} else {
		if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
			if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
				return
			}
		}
	}
	return
}

func (cfg *StcpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	pMsg.Sk = cfg.Sk
}

func (cfg *StcpProxyConf) Check() (err error) {
	return
}

// XTCP
type XtcpProxyConf struct {
	BaseProxyConf

	Role string `json:"role"`
	Sk   string `json:"sk"`

	// used in role server
	LocalSvrConf
	PluginConf

	// used in role visitor
	ServerName string `json:"server_name"`
	BindAddr   string `json:"bind_addr"`
	BindPort   int    `json:"bind_port"`
}

func (cfg *XtcpProxyConf) Compare(cmp ProxyConf) bool {
	cmpConf, ok := cmp.(*XtcpProxyConf)
	if !ok {
		return false
	}

	if !cfg.BaseProxyConf.compare(&cmpConf.BaseProxyConf) ||
		!cfg.LocalSvrConf.compare(&cmpConf.LocalSvrConf) ||
		!cfg.PluginConf.compare(&cmpConf.PluginConf) ||
		cfg.Role != cmpConf.Role ||
		cfg.Sk != cmpConf.Sk ||
		cfg.ServerName != cmpConf.ServerName ||
		cfg.BindAddr != cmpConf.BindAddr ||
		cfg.BindPort != cmpConf.BindPort {
		return false
	}
	return true
}

// Only for role server.
func (cfg *XtcpProxyConf) LoadFromMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.LoadFromMsg(pMsg)
	cfg.Sk = pMsg.Sk
}

func (cfg *XtcpProxyConf) LoadFromFile(name string, section ini.Section) (err error) {
	if err = cfg.BaseProxyConf.LoadFromFile(name, section); err != nil {
		return
	}

	tmpStr := section["role"]
	if tmpStr == "" {
		tmpStr = "server"
	}
	if tmpStr == "server" || tmpStr == "visitor" {
		cfg.Role = tmpStr
	} else {
		return fmt.Errorf("Parse conf error: proxy [%s] incorrect role [%s]", name, tmpStr)
	}

	cfg.Sk = section["sk"]

	if tmpStr == "visitor" {
		prefix := section["prefix"]
		cfg.ServerName = prefix + section["server_name"]
		if cfg.BindAddr = section["bind_addr"]; cfg.BindAddr == "" {
			cfg.BindAddr = "127.0.0.1"
		}

		if tmpStr, ok := section["bind_port"]; ok {
			if cfg.BindPort, err = strconv.Atoi(tmpStr); err != nil {
				return fmt.Errorf("Parse conf error: proxy [%s] bind_port error", name)
			}
		} else {
			return fmt.Errorf("Parse conf error: proxy [%s] bind_port not found", name)
		}
	} else {
		if err = cfg.PluginConf.LoadFromFile(name, section); err != nil {
			if err = cfg.LocalSvrConf.LoadFromFile(name, section); err != nil {
				return
			}
		}
	}
	return
}

func (cfg *XtcpProxyConf) UnMarshalToMsg(pMsg *msg.NewProxy) {
	cfg.BaseProxyConf.UnMarshalToMsg(pMsg)
	pMsg.Sk = cfg.Sk
}

func (cfg *XtcpProxyConf) Check() (err error) {
	return
}

// if len(startProxy) is 0, start all
// otherwise just start proxies in startProxy map
func LoadProxyConfFromFile(prefix string, conf ini.File, startProxy map[string]struct{}) (
	proxyConfs map[string]ProxyConf, visitorConfs map[string]ProxyConf, err error) {

	if prefix != "" {
		prefix += "."
	}

	startAll := true
	if len(startProxy) > 0 {
		startAll = false
	}
	proxyConfs = make(map[string]ProxyConf)
	visitorConfs = make(map[string]ProxyConf)
	for name, section := range conf {
		_, shouldStart := startProxy[name]
		if name != "common" && (startAll || shouldStart) {
			// some proxy or visotr configure may be used this prefix
			section["prefix"] = prefix
			cfg, err := NewProxyConfFromFile(name, section)
			if err != nil {
				return proxyConfs, visitorConfs, err
			}

			role := section["role"]
			if role == "visitor" {
				visitorConfs[prefix+name] = cfg
			} else {
				proxyConfs[prefix+name] = cfg
			}
		}
	}
	return
}
