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

package msg

import "net"

const (
	TypeLogin                 = 'o'
	TypeLoginResp             = '1'
	TypeNewProxy              = 'p'
	TypeNewProxyResp          = '2'
	TypeCloseProxy            = 'c'
	TypeNewWorkConn           = 'w'
	TypeReqWorkConn           = 'r'
	TypeStartWorkConn         = 's'
	TypeNewVisitorConn        = 'v'
	TypeNewVisitorConnResp    = '3'
	TypePing                  = 'h'
	TypePong                  = '4'
	TypeUDPPacket             = 'u'
	TypeNatHoleVisitor        = 'i'
	TypeNatHoleClient         = 'n'
	TypeNatHoleResp           = 'm'
	TypeNatHoleClientDetectOK = 'd'
	TypeNatHoleSid            = '5'
)

var msgTypeMap = map[byte]interface{}{
	TypeLogin:                 Login{},
	TypeLoginResp:             LoginResp{},
	TypeNewProxy:              NewProxy{},
	TypeNewProxyResp:          NewProxyResp{},
	TypeCloseProxy:            CloseProxy{},
	TypeNewWorkConn:           NewWorkConn{},
	TypeReqWorkConn:           ReqWorkConn{},
	TypeStartWorkConn:         StartWorkConn{},
	TypeNewVisitorConn:        NewVisitorConn{},
	TypeNewVisitorConnResp:    NewVisitorConnResp{},
	TypePing:                  Ping{},
	TypePong:                  Pong{},
	TypeUDPPacket:             UDPPacket{},
	TypeNatHoleVisitor:        NatHoleVisitor{},
	TypeNatHoleClient:         NatHoleClient{},
	TypeNatHoleResp:           NatHoleResp{},
	TypeNatHoleClientDetectOK: NatHoleClientDetectOK{},
	TypeNatHoleSid:            NatHoleSid{},
}

// When frpc start, client send this message to login to server.
type Login struct {
	Version      string            `json:"version,omitempty"`
	Hostname     string            `json:"hostname,omitempty"`
	Os           string            `json:"os,omitempty"`
	Arch         string            `json:"arch,omitempty"`
	User         string            `json:"user,omitempty"`
	PrivilegeKey string            `json:"privilege_key,omitempty"`
	Timestamp    int64             `json:"timestamp,omitempty"`
	RunID        string            `json:"run_id,omitempty"`
	Metas        map[string]string `json:"metas,omitempty"`

	// Some global configures.
	PoolCount int `json:"pool_count,omitempty"`
}

type LoginResp struct {
	Version       string `json:"version,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	ServerUDPPort int    `json:"server_udp_port,omitempty"`
	Error         string `json:"error,omitempty"`
}

// When frpc login success, send this message to frps for running a new proxy.
type NewProxy struct {
	ProxyName      string            `json:"proxy_name,omitempty"`
	ProxyType      string            `json:"proxy_type,omitempty"`
	UseEncryption  bool              `json:"use_encryption,omitempty"`
	UseCompression bool              `json:"use_compression,omitempty"`
	Group          string            `json:"group,omitempty"`
	GroupKey       string            `json:"group_key,omitempty"`
	Metas          map[string]string `json:"metas,omitempty"`

	// tcp and udp only
	RemotePort int `json:"remote_port,omitempty"`

	// http and https only
	CustomDomains     []string          `json:"custom_domains,omitempty"`
	SubDomain         string            `json:"subdomain,omitempty"`
	Locations         []string          `json:"locations,omitempty"`
	HTTPUser          string            `json:"http_user,omitempty"`
	HTTPPwd           string            `json:"http_pwd,omitempty"`
	HostHeaderRewrite string            `json:"host_header_rewrite,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	RouteByHTTPUser   string            `json:"route_by_http_user,omitempty"`

	// stcp
	Sk string `json:"sk,omitempty"`

	// tcpmux
	Multiplexer string `json:"multiplexer,omitempty"`
}

type NewProxyResp struct {
	ProxyName  string `json:"proxy_name,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	Error      string `json:"error,omitempty"`
}

type CloseProxy struct {
	ProxyName string `json:"proxy_name,omitempty"`
}

type NewWorkConn struct {
	RunID        string `json:"run_id,omitempty"`
	PrivilegeKey string `json:"privilege_key,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
}

type ReqWorkConn struct{}

type StartWorkConn struct {
	ProxyName string `json:"proxy_name,omitempty"`
	SrcAddr   string `json:"src_addr,omitempty"`
	DstAddr   string `json:"dst_addr,omitempty"`
	SrcPort   uint16 `json:"src_port,omitempty"`
	DstPort   uint16 `json:"dst_port,omitempty"`
	Error     string `json:"error,omitempty"`
}

type NewVisitorConn struct {
	ProxyName      string `json:"proxy_name,omitempty"`
	SignKey        string `json:"sign_key,omitempty"`
	Timestamp      int64  `json:"timestamp,omitempty"`
	UseEncryption  bool   `json:"use_encryption,omitempty"`
	UseCompression bool   `json:"use_compression,omitempty"`
}

type NewVisitorConnResp struct {
	ProxyName string `json:"proxy_name,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Ping struct {
	PrivilegeKey string `json:"privilege_key,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
}

type Pong struct {
	Error string `json:"error,omitempty"`
}

type UDPPacket struct {
	Content    string       `json:"c,omitempty"`
	LocalAddr  *net.UDPAddr `json:"l,omitempty"`
	RemoteAddr *net.UDPAddr `json:"r,omitempty"`
}

type NatHoleVisitor struct {
	ProxyName string `json:"proxy_name,omitempty"`
	SignKey   string `json:"sign_key,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type NatHoleClient struct {
	ProxyName string `json:"proxy_name,omitempty"`
	Sid       string `json:"sid,omitempty"`
}

type NatHoleResp struct {
	Sid         string `json:"sid,omitempty"`
	VisitorAddr string `json:"visitor_addr,omitempty"`
	ClientAddr  string `json:"client_addr,omitempty"`
	Error       string `json:"error,omitempty"`
}

type NatHoleClientDetectOK struct{}

type NatHoleSid struct {
	Sid string `json:"sid,omitempty"`
}
