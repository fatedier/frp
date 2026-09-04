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
	AutoTransportVersion uint32 = 1

	TypeLogin              byte = 'o'
	TypeLoginResp          byte = '1'
	TypeNewProxy           byte = 'p'
	TypeNewProxyResp       byte = '2'
	TypeCloseProxy         byte = 'c'
	TypeNewWorkConn        byte = 'w'
	TypeReqWorkConn        byte = 'r'
	TypeStartWorkConn      byte = 's'
	TypeNewVisitorConn     byte = 'v'
	TypeNewVisitorConnResp byte = '3'
	TypePing               byte = 'h'
	TypePong               byte = '4'
	TypeUDPPacket          byte = 'u'
	TypeNatHoleVisitor     byte = 'i'
	TypeNatHoleClient      byte = 'n'
	TypeNatHoleResp        byte = 'm'
	TypeNatHoleSid         byte = '5'
	TypeNatHoleReport      byte = '6'
	TypeClientHelloAuto    byte = 'a'
	TypeServerHelloAuto    byte = 'b'
	TypeSelectTransport    byte = 'd'
	TypeProbeTransport     byte = 'e'
	TypeProbeTransportResp byte = 'f'
)

var msgTypeMap = map[byte]any{
	TypeLogin:              Login{},
	TypeLoginResp:          LoginResp{},
	TypeNewProxy:           NewProxy{},
	TypeNewProxyResp:       NewProxyResp{},
	TypeCloseProxy:         CloseProxy{},
	TypeNewWorkConn:        NewWorkConn{},
	TypeReqWorkConn:        ReqWorkConn{},
	TypeStartWorkConn:      StartWorkConn{},
	TypeNewVisitorConn:     NewVisitorConn{},
	TypeNewVisitorConnResp: NewVisitorConnResp{},
	TypePing:               Ping{},
	TypePong:               Pong{},
	TypeUDPPacket:          UDPPacket{},
	TypeNatHoleVisitor:     NatHoleVisitor{},
	TypeNatHoleClient:      NatHoleClient{},
	TypeNatHoleResp:        NatHoleResp{},
	TypeNatHoleSid:         NatHoleSid{},
	TypeNatHoleReport:      NatHoleReport{},
	TypeClientHelloAuto:    ClientHelloAuto{},
	TypeServerHelloAuto:    ServerHelloAuto{},
	TypeSelectTransport:    SelectTransport{},
	TypeProbeTransport:     ProbeTransport{},
	TypeProbeTransportResp: ProbeTransportResp{},
}

var TypeNameNatHoleResp = reflect.TypeFor[NatHoleResp]().Name()

type ClientSpec struct {
	// Due to the support of VirtualClient, frps needs to know the client type in order to
	// differentiate the processing logic.
	// Optional values: ssh-tunnel
	Type string `json:"type,omitempty"`
	// If the value is true, the client will not require authentication.
	AlwaysAuthPass bool `json:"always_auth_pass,omitempty"`
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
	ClientID     string            `json:"client_id,omitempty"`
	Metas        map[string]string `json:"metas,omitempty"`

	// Currently only effective for VirtualClient.
	ClientSpec ClientSpec `json:"client_spec,omitempty"`

	// Some global configures.
	PoolCount int `json:"pool_count,omitempty"`
}

type LoginResp struct {
	Version string `json:"version,omitempty"`
	RunID   string `json:"run_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ClientHelloAuto struct {
	ProtocolMode       string   `json:"protocol_mode,omitempty"`
	ClientCandidates   []string `json:"client_candidates,omitempty"`
	AllowUDP           bool     `json:"allow_udp,omitempty"`
	HasProxyURL        bool     `json:"has_proxy_url,omitempty"`
	TLSRequired        bool     `json:"tls_required,omitempty"`
	Strategy           string   `json:"strategy,omitempty"`
	LastGoodProtocol   string   `json:"last_good_protocol,omitempty"`
	BlacklistProtocols []string `json:"blacklist_protocols,omitempty"`
	ClientAutoVersion  uint32   `json:"client_auto_version,omitempty"`
	PrivilegeKey       string   `json:"privilege_key,omitempty"`
	Timestamp          int64    `json:"timestamp,omitempty"`
	Login              *Login   `json:"login,omitempty"`
}

type ServerHelloAuto struct {
	ProtocolMode       string              `json:"protocol_mode,omitempty"`
	AutoEnabled        bool                `json:"auto_enabled,omitempty"`
	AllowDynamicSwitch bool                `json:"allow_dynamic_switch,omitempty"`
	PreferOrder        []string            `json:"prefer_order,omitempty"`
	Transports         []TransportEndpoint `json:"transports,omitempty"`
	ServerAutoVersion  uint32              `json:"server_auto_version,omitempty"`
	Error              string              `json:"error,omitempty"`
}

type TransportEndpoint struct {
	Protocol string `json:"protocol,omitempty"`
	Addr     string `json:"addr,omitempty"`
	Port     int    `json:"port,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
}

type SelectTransport struct {
	Protocol          string           `json:"protocol,omitempty"`
	Addr              string           `json:"addr,omitempty"`
	Port              int              `json:"port,omitempty"`
	Reason            string           `json:"reason,omitempty"`
	Scores            map[string]int64 `json:"scores,omitempty"`
	ClientAutoVersion uint32           `json:"client_auto_version,omitempty"`
}

type ProbeTransport struct {
	Protocol          string `json:"protocol,omitempty"`
	Addr              string `json:"addr,omitempty"`
	Port              int    `json:"port,omitempty"`
	ClientAutoVersion uint32 `json:"client_auto_version,omitempty"`
	PrivilegeKey      string `json:"privilege_key,omitempty"`
	Timestamp         int64  `json:"timestamp,omitempty"`
	Login             *Login `json:"login,omitempty"`
}

type ProbeTransportResp struct {
	Protocol          string `json:"protocol,omitempty"`
	Port              int    `json:"port,omitempty"`
	ServerAutoVersion uint32 `json:"server_auto_version,omitempty"`
	Error             string `json:"error,omitempty"`
}

// When frpc login success, send this message to frps for running a new proxy.
type NewProxy struct {
	ProxyName          string            `json:"proxy_name,omitempty"`
	ProxyType          string            `json:"proxy_type,omitempty"`
	UseEncryption      bool              `json:"use_encryption,omitempty"`
	UseCompression     bool              `json:"use_compression,omitempty"`
	BandwidthLimit     string            `json:"bandwidth_limit,omitempty"`
	BandwidthLimitMode string            `json:"bandwidth_limit_mode,omitempty"`
	Group              string            `json:"group,omitempty"`
	GroupKey           string            `json:"group_key,omitempty"`
	Metas              map[string]string `json:"metas,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`

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
	ResponseHeaders   map[string]string `json:"response_headers,omitempty"`
	RouteByHTTPUser   string            `json:"route_by_http_user,omitempty"`

	// stcp, sudp, xtcp
	Sk         string   `json:"sk,omitempty"`
	AllowUsers []string `json:"allow_users,omitempty"`

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
	RunID          string `json:"run_id,omitempty"`
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
	Content    []byte       `json:"c,omitempty"`
	LocalAddr  *net.UDPAddr `json:"l,omitempty"`
	RemoteAddr *net.UDPAddr `json:"r,omitempty"`
}

type NatHoleVisitor struct {
	TransactionID string   `json:"transaction_id,omitempty"`
	ProxyName     string   `json:"proxy_name,omitempty"`
	PreCheck      bool     `json:"pre_check,omitempty"`
	Protocol      string   `json:"protocol,omitempty"`
	SignKey       string   `json:"sign_key,omitempty"`
	Timestamp     int64    `json:"timestamp,omitempty"`
	MappedAddrs   []string `json:"mapped_addrs,omitempty"`
	AssistedAddrs []string `json:"assisted_addrs,omitempty"`
}

type NatHoleClient struct {
	TransactionID string   `json:"transaction_id,omitempty"`
	ProxyName     string   `json:"proxy_name,omitempty"`
	Sid           string   `json:"sid,omitempty"`
	MappedAddrs   []string `json:"mapped_addrs,omitempty"`
	AssistedAddrs []string `json:"assisted_addrs,omitempty"`
}

type PortsRange struct {
	From int `json:"from,omitempty"`
	To   int `json:"to,omitempty"`
}

type NatHoleDetectBehavior struct {
	Role              string       `json:"role,omitempty"` // sender or receiver
	Mode              int          `json:"mode,omitempty"` // 0, 1, 2...
	TTL               int          `json:"ttl,omitempty"`
	SendDelayMs       int          `json:"send_delay_ms,omitempty"`
	ReadTimeoutMs     int          `json:"read_timeout,omitempty"`
	CandidatePorts    []PortsRange `json:"candidate_ports,omitempty"`
	SendRandomPorts   int          `json:"send_random_ports,omitempty"`
	ListenRandomPorts int          `json:"listen_random_ports,omitempty"`
}

type NatHoleResp struct {
	TransactionID  string                `json:"transaction_id,omitempty"`
	Sid            string                `json:"sid,omitempty"`
	Protocol       string                `json:"protocol,omitempty"`
	CandidateAddrs []string              `json:"candidate_addrs,omitempty"`
	AssistedAddrs  []string              `json:"assisted_addrs,omitempty"`
	DetectBehavior NatHoleDetectBehavior `json:"detect_behavior,omitempty"`
	Error          string                `json:"error,omitempty"`
}

type NatHoleSid struct {
	TransactionID string `json:"transaction_id,omitempty"`
	Sid           string `json:"sid,omitempty"`
	Response      bool   `json:"response,omitempty"`
	Nonce         string `json:"nonce,omitempty"`
}

type NatHoleReport struct {
	Sid     string `json:"sid,omitempty"`
	Success bool   `json:"success,omitempty"`
}
