// Copyright 2020 The frp Authors
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
	"os"
	"strings"

	"github.com/fatedier/frp/pkg/auth"
	"github.com/fatedier/frp/pkg/util/util"

	"gopkg.in/ini.v1"
)

// ClientCommonConf contains information for a client service. It is
// recommended to use GetDefaultClientConf instead of creating this object
// directly, so that all unspecified fields have reasonable default values.
type ClientCommonConf struct {
	auth.ClientConfig `ini:",extends"`

	// ServerAddr specifies the address of the server to connect to. By
	// default, this value is "0.0.0.0".
	ServerAddr string `ini:"server_addr" josn:"server_addr"`
	// ServerPort specifies the port to connect to the server on. By default,
	// this value is 7000.
	ServerPort int `ini:"server_port" json:"server_port"`
	// HTTPProxy specifies a proxy address to connect to the server through. If
	// this value is "", the server will be connected to directly. By default,
	// this value is read from the "http_proxy" environment variable.
	HTTPProxy string `ini:"http_proxy" json:"http_proxy"`
	// LogFile specifies a file where logs will be written to. This value will
	// only be used if LogWay is set appropriately. By default, this value is
	// "console".
	LogFile string `ini:"log_file" json:"log_file"`
	// LogWay specifies the way logging is managed. Valid values are "console"
	// or "file". If "console" is used, logs will be printed to stdout. If
	// "file" is used, logs will be printed to LogFile. By default, this value
	// is "console".
	LogWay string `ini:"log_way" json:"log_way"`
	// LogLevel specifies the minimum log level. Valid values are "trace",
	// "debug", "info", "warn", and "error". By default, this value is "info".
	LogLevel string `ini:"log_level" json:"log_level"`
	// LogMaxDays specifies the maximum number of days to store log information
	// before deletion. This is only used if LogWay == "file". By default, this
	// value is 0.
	LogMaxDays int64 `ini:"log_max_days" json:"log_max_days"`
	// DisableLogColor disables log colors when LogWay == "console" when set to
	// true. By default, this value is false.
	DisableLogColor bool `ini:"disable_log_color" json:"disable_log_color"`
	// AdminAddr specifies the address that the admin server binds to. By
	// default, this value is "127.0.0.1".
	AdminAddr string `ini:"admin_addr" json:"admin_addr"`
	// AdminPort specifies the port for the admin server to listen on. If this
	// value is 0, the admin server will not be started. By default, this value
	// is 0.
	AdminPort int `ini:"admin_port" json:"admin_port"`
	// AdminUser specifies the username that the admin server will use for
	// login. By default, this value is "admin".
	AdminUser string `ini:"admin_user" json:"admin_user"`
	// AdminPwd specifies the password that the admin server will use for
	// login. By default, this value is "admin".
	AdminPwd string `ini:"admin_pwd" json:"admin_pwd"`
	// AssetsDir specifies the local directory that the admin server will load
	// resources from. If this value is "", assets will be loaded from the
	// bundled executable using statik. By default, this value is "".
	AssetsDir string `ini:"assets_dir" json:"assets_dir"`
	// PoolCount specifies the number of connections the client will make to
	// the server in advance. By default, this value is 0.
	PoolCount int `ini:"pool_count" json:"pool_count"`
	// TCPMux toggles TCP stream multiplexing. This allows multiple requests
	// from a client to share a single TCP connection. If this value is true,
	// the server must have TCP multiplexing enabled as well. By default, this
	// value is true.
	TCPMux bool `ini:"tcp_mux" json:"tcp_mux"`
	// User specifies a prefix for proxy names to distinguish them from other
	// clients. If this value is not "", proxy names will automatically be
	// changed to "{user}.{proxy_name}". By default, this value is "".
	User string `ini:"user" json:"user"`
	// DNSServer specifies a DNS server address for FRPC to use. If this value
	// is "", the default DNS will be used. By default, this value is "".
	DNSServer string `ini:"dns_server" json:"dns_server"`
	// LoginFailExit controls whether or not the client should exit after a
	// failed login attempt. If false, the client will retry until a login
	// attempt succeeds. By default, this value is true.
	LoginFailExit bool `ini:"login_fail_exit" json:"login_fail_exit"`
	// Start specifies a set of enabled proxies by name. If this set is empty,
	// all supplied proxies are enabled. By default, this value is an empty
	// set.
	Start []string `ini:"start" json:"start"`
	//Start map[string]struct{} `json:"start"`
	// Protocol specifies the protocol to use when interacting with the server.
	// Valid values are "tcp", "kcp" and "websocket". By default, this value
	// is "tcp".
	Protocol string `ini:"protocol" json:"protocol"`
	// TLSEnable specifies whether or not TLS should be used when communicating
	// with the server. If "tls_cert_file" and "tls_key_file" are valid,
	// client will load the supplied tls configuration.
	TLSEnable bool `ini:"tls_enable" json:"tls_enable"`
	// TLSCertPath specifies the path of the cert file that client will
	// load. It only works when "tls_enable" is true and "tls_key_file" is valid.
	TLSCertFile string `ini:"tls_cert_file" json:"tls_cert_file"`
	// TLSKeyPath specifies the path of the secret key file that client
	// will load. It only works when "tls_enable" is true and "tls_cert_file"
	// are valid.
	TLSKeyFile string `ini:"tls_key_file" json:"tls_key_file"`
	// TLSTrustedCaFile specifies the path of the trusted ca file that will load.
	// It only works when "tls_enable" is valid and tls configuration of server
	// has been specified.
	TLSTrustedCaFile string `ini:"tls_trusted_ca_file" json:"tls_trusted_ca_file"`
	// TLSServerName specifices the custom server name of tls certificate. By
	// default, server name if same to ServerAddr.
	TLSServerName string `ini:"tls_server_name" json:"tls_server_name"`
	// HeartBeatInterval specifies at what interval heartbeats are sent to the
	// server, in seconds. It is not recommended to change this value. By
	// default, this value is 30.
	HeartbeatInterval int64 `ini:"heartbeat_interval" json:"heartbeat_interval"`
	// HeartBeatTimeout specifies the maximum allowed heartbeat response delay
	// before the connection is terminated, in seconds. It is not recommended
	// to change this value. By default, this value is 90.
	HeartbeatTimeout int64 `ini:"heartbeat_timeout" json:"heartbeat_timeout"`
	// Client meta info
	Metas map[string]string `ini:"-" json:"metas"`
	// UDPPacketSize specifies the udp packet size
	// By default, this value is 1500
	UDPPacketSize int64 `ini:"udp_packet_size" json:"udp_packet_size"`
}

// GetDefaultClientConf returns a client configuration with default values.
func GetDefaultClientConf() ClientCommonConf {
	return ClientCommonConf{
		ClientConfig:      auth.GetDefaultClientConf(),
		ServerAddr:        "0.0.0.0",
		ServerPort:        7000,
		HTTPProxy:         os.Getenv("http_proxy"),
		LogFile:           "console",
		LogWay:            "console",
		LogLevel:          "info",
		LogMaxDays:        3,
		DisableLogColor:   false,
		AdminAddr:         "127.0.0.1",
		AdminPort:         0,
		AdminUser:         "",
		AdminPwd:          "",
		AssetsDir:         "",
		PoolCount:         1,
		TCPMux:            true,
		User:              "",
		DNSServer:         "",
		LoginFailExit:     true,
		Start:             make([]string, 0),
		Protocol:          "tcp",
		TLSEnable:         false,
		TLSCertFile:       "",
		TLSKeyFile:        "",
		TLSTrustedCaFile:  "",
		HeartbeatInterval: 30,
		HeartbeatTimeout:  90,
		Metas:             make(map[string]string),
		UDPPacketSize:     1500,
	}
}

func (cfg *ClientCommonConf) Complete() {
	if cfg.LogFile == "console" {
		cfg.LogWay = "console"
	} else {
		cfg.LogWay = "file"
	}
}

func (cfg *ClientCommonConf) Validate() error {
	if cfg.HeartbeatInterval <= 0 {
		return fmt.Errorf("invalid heartbeat_interval")
	}

	if cfg.HeartbeatTimeout < cfg.HeartbeatInterval {
		return fmt.Errorf("invalid heartbeat_timeout, heartbeat_timeout is less than heartbeat_interval")
	}

	if cfg.TLSEnable == false {
		if cfg.TLSCertFile != "" {
			fmt.Println("WARNING! tls_cert_file is invalid when tls_enable is false")
		}

		if cfg.TLSKeyFile != "" {
			fmt.Println("WARNING! tls_key_file is invalid when tls_enable is false")
		}

		if cfg.TLSTrustedCaFile != "" {
			fmt.Println("WARNING! tls_trusted_ca_file is invalid when tls_enable is false")
		}
	}

	if cfg.Protocol != "tcp" && cfg.Protocol != "kcp" && cfg.Protocol != "websocket" {
		return fmt.Errorf("invalid protocol")
	}

	return nil
}

// Supported sources including: string(file path), []byte, Reader interface.
func UnmarshalClientConfFromIni(source interface{}) (ClientCommonConf, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return ClientCommonConf{}, err
	}

	s, err := f.GetSection("common")
	if err != nil {
		return ClientCommonConf{}, fmt.Errorf("invalid configuration file, not found [common] section")
	}

	common := GetDefaultClientConf()
	err = s.MapTo(&common)
	if err != nil {
		return ClientCommonConf{}, err
	}

	common.Metas = GetMapWithoutPrefix(s.KeysHash(), "meta_")

	return common, nil
}

// if len(startProxy) is 0, start all
// otherwise just start proxies in startProxy map
func LoadAllProxyConfsFromIni(
	prefix string,
	source interface{},
	start []string,
) (map[string]ProxyConf, map[string]VisitorConf, error) {

	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         false,
		InsensitiveSections: false,
		InsensitiveKeys:     false,
		IgnoreInlineComment: true,
		AllowBooleanKeys:    true,
	}, source)
	if err != nil {
		return nil, nil, err
	}

	proxyConfs := make(map[string]ProxyConf)
	visitorConfs := make(map[string]VisitorConf)

	if prefix != "" {
		prefix += "."
	}

	startProxy := make(map[string]struct{})
	for _, s := range start {
		startProxy[s] = struct{}{}
	}

	startAll := true
	if len(startProxy) > 0 {
		startAll = false
	}

	// Build template sections from range section And append to ini.File.
	rangeSections := make([]*ini.Section, 0)
	for _, section := range f.Sections() {

		if !strings.HasPrefix(section.Name(), "range:") {
			continue
		}

		rangeSections = append(rangeSections, section)
	}

	for _, section := range rangeSections {
		err = renderRangeProxyTemplates(f, section)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to render range-section[%s] with error: %v", section.Name(), err)
		}
	}

	for _, section := range f.Sections() {
		name := section.Name()

		if name == ini.DefaultSection || name == "common" || strings.HasPrefix(name, "range:") {
			continue
		}

		_, shouldStart := startProxy[name]
		if !startAll && !shouldStart {
			continue
		}

		roleType := section.Key("role").String()
		if roleType == "" {
			roleType = "server"
		}

		switch roleType {
		case "server":
			newConf, newErr := NewProxyConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, fmt.Errorf("fail to parse section[%s], err: %v", name, newErr)
			}
			proxyConfs[prefix+name] = newConf
		case "visitor":
			newConf, newErr := NewVisitorConfFromIni(prefix, name, section)
			if newErr != nil {
				return nil, nil, newErr
			}
			visitorConfs[prefix+name] = newConf
		default:
			return nil, nil, fmt.Errorf("section[%s] role should be 'server' or 'visitor'", name)
		}
	}
	return proxyConfs, visitorConfs, nil
}

func renderRangeProxyTemplates(f *ini.File, section *ini.Section) error {

	// Validation
	localPortStr := section.Key("local_port").String()
	remotePortStr := section.Key("remote_port").String()
	if localPortStr == "" || remotePortStr == "" {
		return fmt.Errorf("local_port or remote_port is empty")
	}

	localPorts, err := util.ParseRangeNumbers(localPortStr)
	if err != nil {
		return err
	}

	remotePorts, err := util.ParseRangeNumbers(remotePortStr)
	if err != nil {
		return err
	}

	if len(localPorts) != len(remotePorts) {
		return fmt.Errorf("local ports number should be same with remote ports number")
	}

	if len(localPorts) == 0 {
		return fmt.Errorf("local_port and remote_port is necessary")
	}

	// Templates
	prefix := strings.TrimSpace(strings.TrimPrefix(section.Name(), "range:"))

	for i := range localPorts {
		tmpname := fmt.Sprintf("%s_%d", prefix, i)

		tmpsection, err := f.NewSection(tmpname)
		if err != nil {
			return err
		}

		copySection(section, tmpsection)
		tmpsection.NewKey("local_port", fmt.Sprintf("%d", localPorts[i]))
		tmpsection.NewKey("remote_port", fmt.Sprintf("%d", remotePorts[i]))
	}

	return nil
}

func copySection(source, target *ini.Section) {
	for key, value := range source.KeysHash() {
		target.NewKey(key, value)
	}
}
