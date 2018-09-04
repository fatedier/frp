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

import (
	"net"
	"reflect"
)

const (
	TypeLogin              = 'o'
	TypeLoginResp          = '1'
	TypeNewProxy           = 'p'
	TypeNewProxyResp       = '2'
	TypeCloseProxy         = 'c'
	TypeNewWorkConn        = 'w'
	TypeReqWorkConn        = 'r'
	TypeStartWorkConn      = 's'
	TypeNewVisitorConn     = 'v'
	TypeNewVisitorConnResp = '3'
	TypePing               = 'h'
	TypePong               = '4'
	TypeUdpPacket          = 'u'
	TypeNatHoleVisitor     = 'i'
	TypeNatHoleClient      = 'n'
	TypeNatHoleResp        = 'm'
	TypeNatHoleSid         = '5'
)

var (
	TypeMap       map[byte]reflect.Type
	TypeStringMap map[reflect.Type]byte
)

func init() {
	TypeMap = make(map[byte]reflect.Type)
	TypeStringMap = make(map[reflect.Type]byte)

	TypeMap[TypeLogin] = reflect.TypeOf(Login{})
	TypeMap[TypeLoginResp] = reflect.TypeOf(LoginResp{})
	TypeMap[TypeNewProxy] = reflect.TypeOf(NewProxy{})
	TypeMap[TypeNewProxyResp] = reflect.TypeOf(NewProxyResp{})
	TypeMap[TypeCloseProxy] = reflect.TypeOf(CloseProxy{})
	TypeMap[TypeNewWorkConn] = reflect.TypeOf(NewWorkConn{})
	TypeMap[TypeReqWorkConn] = reflect.TypeOf(ReqWorkConn{})
	TypeMap[TypeStartWorkConn] = reflect.TypeOf(StartWorkConn{})
	TypeMap[TypeNewVisitorConn] = reflect.TypeOf(NewVisitorConn{})
	TypeMap[TypeNewVisitorConnResp] = reflect.TypeOf(NewVisitorConnResp{})
	TypeMap[TypePing] = reflect.TypeOf(Ping{})
	TypeMap[TypePong] = reflect.TypeOf(Pong{})
	TypeMap[TypeUdpPacket] = reflect.TypeOf(UdpPacket{})
	TypeMap[TypeNatHoleVisitor] = reflect.TypeOf(NatHoleVisitor{})
	TypeMap[TypeNatHoleClient] = reflect.TypeOf(NatHoleClient{})
	TypeMap[TypeNatHoleResp] = reflect.TypeOf(NatHoleResp{})
	TypeMap[TypeNatHoleSid] = reflect.TypeOf(NatHoleSid{})

	for k, v := range TypeMap {
		TypeStringMap[v] = k
	}
}

// Message wraps socket packages for communicating between frpc and frps.
type Message interface{}

// When frpc start, client send this message to login to server.
type Login struct {
	Version      string `json:"version"`
	Hostname     string `json:"hostname"`
	Os           string `json:"os"`
	Arch         string `json:"arch"`
	User         string `json:"user"`
	PrivilegeKey string `json:"privilege_key"`
	Timestamp    int64  `json:"timestamp"`
	RunId        string `json:"run_id"`

	// Some global configures.
	PoolCount int `json:"pool_count"`
}

type LoginResp struct {
	Version       string `json:"version"`
	RunId         string `json:"run_id"`
	ServerUdpPort int    `json:"server_udp_port"`
	Error         string `json:"error"`
}

// When frpc login success, send this message to frps for running a new proxy.
type NewProxy struct {
	ProxyName      string `json:"proxy_name"`
	ProxyType      string `json:"proxy_type"`
	UseEncryption  bool   `json:"use_encryption"`
	UseCompression bool   `json:"use_compression"`

	// tcp and udp only
	RemotePort int `json:"remote_port"`

	// http and https only
	CustomDomains     []string `json:"custom_domains"`
	SubDomain         string   `json:"subdomain"`
	Locations         []string `json:"locations"`
	HostHeaderRewrite string   `json:"host_header_rewrite"`
	HttpUser          string   `json:"http_user"`
	HttpPwd           string   `json:"http_pwd"`

	// stcp
	Sk string `json:"sk"`
}

type NewProxyResp struct {
	ProxyName  string `json:"proxy_name"`
	RemoteAddr string `json:"remote_addr"`
	Error      string `json:"error"`
}

type CloseProxy struct {
	ProxyName string `json:"proxy_name"`
}

type NewWorkConn struct {
	RunId string `json:"run_id"`
}

type ReqWorkConn struct {
}

type StartWorkConn struct {
	ProxyName string `json:"proxy_name"`
}

type NewVisitorConn struct {
	ProxyName      string `json:"proxy_name"`
	SignKey        string `json:"sign_key"`
	Timestamp      int64  `json:"timestamp"`
	UseEncryption  bool   `json:"use_encryption"`
	UseCompression bool   `json:"use_compression"`
}

type NewVisitorConnResp struct {
	ProxyName string `json:"proxy_name"`
	Error     string `json:"error"`
}

type Ping struct {
}

type Pong struct {
}

type UdpPacket struct {
	Content    string       `json:"c"`
	LocalAddr  *net.UDPAddr `json:"l"`
	RemoteAddr *net.UDPAddr `json:"r"`
}

type NatHoleVisitor struct {
	ProxyName string `json:"proxy_name"`
	SignKey   string `json:"sign_key"`
	Timestamp int64  `json:"timestamp"`
}

type NatHoleClient struct {
	ProxyName string `json:"proxy_name"`
	Sid       string `json:"sid"`
}

type NatHoleResp struct {
	Sid         string `json:"sid"`
	VisitorAddr string `json:"visitor_addr"`
	ClientAddr  string `json:"client_addr"`
}

type NatHoleSid struct {
	Sid string `json:"sid"`
}
