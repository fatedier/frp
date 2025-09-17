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

package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
)

// WordSepNormalizeFunc changes all flags that contain "_" separators
func WordSepNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if strings.Contains(name, "_") {
		return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
	}
	return pflag.NormalizedName(name)
}

type RegisterFlagOption func(*registerFlagOptions)

type registerFlagOptions struct {
	sshMode bool
}

func WithSSHMode() RegisterFlagOption {
	return func(o *registerFlagOptions) {
		o.sshMode = true
	}
}

type BandwidthQuantityFlag struct {
	V *types.BandwidthQuantity
}

func (f *BandwidthQuantityFlag) Set(s string) error {
	return f.V.UnmarshalString(s)
}

func (f *BandwidthQuantityFlag) String() string {
	return f.V.String()
}

func (f *BandwidthQuantityFlag) Type() string {
	return "string"
}

func RegisterProxyFlags(cmd *cobra.Command, c v1.ProxyConfigurer, opts ...RegisterFlagOption) {
	registerProxyBaseConfigFlags(cmd, c.GetBaseConfig(), opts...)

	switch cc := c.(type) {
	case *v1.TCPProxyConfig:
		cmd.Flags().IntVarP(&cc.RemotePort, "remote_port", "r", 0, "remote port")
	case *v1.UDPProxyConfig:
		cmd.Flags().IntVarP(&cc.RemotePort, "remote_port", "r", 0, "remote port")
	case *v1.HTTPProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
		cmd.Flags().StringSliceVarP(&cc.Locations, "locations", "", []string{}, "locations")
		cmd.Flags().StringVarP(&cc.HTTPUser, "http_user", "", "", "http auth user")
		cmd.Flags().StringVarP(&cc.HTTPPassword, "http_pwd", "", "", "http auth password")
		cmd.Flags().StringVarP(&cc.HostHeaderRewrite, "host_header_rewrite", "", "", "host header rewrite")
	case *v1.HTTPSProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
	case *v1.TCPMuxProxyConfig:
		registerProxyDomainConfigFlags(cmd, &cc.DomainConfig)
		cmd.Flags().StringVarP(&cc.Multiplexer, "mux", "", "", "multiplexer")
		cmd.Flags().StringVarP(&cc.HTTPUser, "http_user", "", "", "http auth user")
		cmd.Flags().StringVarP(&cc.HTTPPassword, "http_pwd", "", "", "http auth password")
	case *v1.STCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "allow visitor users")
	case *v1.SUDPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "allow visitor users")
	case *v1.XTCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
		cmd.Flags().StringSliceVarP(&cc.AllowUsers, "allow_users", "", []string{}, "allow visitor users")
	}
}

func registerProxyBaseConfigFlags(cmd *cobra.Command, c *v1.ProxyBaseConfig, opts ...RegisterFlagOption) {
	if c == nil {
		return
	}
	options := &registerFlagOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cmd.Flags().StringVarP(&c.Name, "proxy_name", "n", "", "proxy name")
	cmd.Flags().StringToStringVarP(&c.Metadatas, "metadatas", "", nil, "metadata key-value pairs (e.g., key1=value1,key2=value2)")
	cmd.Flags().StringToStringVarP(&c.Annotations, "annotations", "", nil, "annotation key-value pairs (e.g., key1=value1,key2=value2)")

	if !options.sshMode {
		cmd.Flags().StringVarP(&c.LocalIP, "local_ip", "i", "127.0.0.1", "local ip")
		cmd.Flags().IntVarP(&c.LocalPort, "local_port", "l", 0, "local port")
		cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
		cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
		cmd.Flags().StringVarP(&c.Transport.BandwidthLimitMode, "bandwidth_limit_mode", "", types.BandwidthLimitModeClient, "bandwidth limit mode")
		cmd.Flags().VarP(&BandwidthQuantityFlag{V: &c.Transport.BandwidthLimit}, "bandwidth_limit", "", "bandwidth limit (e.g. 100KB or 1MB)")
	}
}

func registerProxyDomainConfigFlags(cmd *cobra.Command, c *v1.DomainConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringSliceVarP(&c.CustomDomains, "custom_domain", "d", []string{}, "custom domains")
	cmd.Flags().StringVarP(&c.SubDomain, "sd", "", "", "sub domain")
}

func RegisterVisitorFlags(cmd *cobra.Command, c v1.VisitorConfigurer, opts ...RegisterFlagOption) {
	registerVisitorBaseConfigFlags(cmd, c.GetBaseConfig(), opts...)

	// add visitor flags if exist
}

func registerVisitorBaseConfigFlags(cmd *cobra.Command, c *v1.VisitorBaseConfig, _ ...RegisterFlagOption) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "visitor_name", "n", "", "visitor name")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
	cmd.Flags().StringVarP(&c.SecretKey, "sk", "", "", "secret key")
	cmd.Flags().StringVarP(&c.ServerName, "server_name", "", "", "server name")
	cmd.Flags().StringVarP(&c.ServerUser, "server-user", "", "", "server user")
	cmd.Flags().StringVarP(&c.BindAddr, "bind_addr", "", "", "bind addr")
	cmd.Flags().IntVarP(&c.BindPort, "bind_port", "", 0, "bind port")
}

func RegisterClientCommonConfigFlags(cmd *cobra.Command, c *v1.ClientCommonConfig, opts ...RegisterFlagOption) {
	options := &registerFlagOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if !options.sshMode {
		cmd.PersistentFlags().StringVarP(&c.ServerAddr, "server_addr", "s", "127.0.0.1", "frp server's address")
		cmd.PersistentFlags().IntVarP(&c.ServerPort, "server_port", "P", 7000, "frp server's port")
		cmd.PersistentFlags().StringVarP(&c.Transport.Protocol, "protocol", "p", "tcp",
			fmt.Sprintf("optional values are %v", validation.SupportedTransportProtocols))
		cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "log level")
		cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "console or file path")
		cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "log file reversed days")
		cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "disable log color in console")
		cmd.PersistentFlags().StringVarP(&c.Transport.TLS.ServerName, "tls_server_name", "", "", "specify the custom server name of tls certificate")
		cmd.PersistentFlags().StringVarP(&c.DNSServer, "dns_server", "", "", "specify dns server instead of using system default one")
		c.Transport.TLS.Enable = cmd.PersistentFlags().BoolP("tls_enable", "", true, "enable frpc tls")
	}
	cmd.PersistentFlags().StringVarP(&c.User, "user", "u", "", "user")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "auth token")
}

type PortsRangeSliceFlag struct {
	V *[]types.PortsRange
}

func (f *PortsRangeSliceFlag) String() string {
	if f.V == nil {
		return ""
	}
	return types.PortsRangeSlice(*f.V).String()
}

func (f *PortsRangeSliceFlag) Set(s string) error {
	slice, err := types.NewPortsRangeSliceFromString(s)
	if err != nil {
		return err
	}
	*f.V = slice
	return nil
}

func (f *PortsRangeSliceFlag) Type() string {
	return "string"
}

type BoolFuncFlag struct {
	TrueFunc  func()
	FalseFunc func()

	v bool
}

func (f *BoolFuncFlag) String() string {
	return strconv.FormatBool(f.v)
}

func (f *BoolFuncFlag) Set(s string) error {
	f.v = strconv.FormatBool(f.v) == "true"

	if !f.v {
		if f.FalseFunc != nil {
			f.FalseFunc()
		}
		return nil
	}

	if f.TrueFunc != nil {
		f.TrueFunc()
	}
	return nil
}

func (f *BoolFuncFlag) Type() string {
	return "bool"
}

func RegisterServerConfigFlags(cmd *cobra.Command, c *v1.ServerConfig, opts ...RegisterFlagOption) {
	cmd.PersistentFlags().StringVarP(&c.BindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	cmd.PersistentFlags().IntVarP(&c.BindPort, "bind_port", "p", 7000, "bind port")
	cmd.PersistentFlags().IntVarP(&c.KCPBindPort, "kcp_bind_port", "", 0, "kcp bind udp port")
	cmd.PersistentFlags().IntVarP(&c.QUICBindPort, "quic_bind_port", "", 0, "quic bind udp port")
	cmd.PersistentFlags().StringVarP(&c.ProxyBindAddr, "proxy_bind_addr", "", "0.0.0.0", "proxy bind address")
	cmd.PersistentFlags().IntVarP(&c.VhostHTTPPort, "vhost_http_port", "", 0, "vhost http port")
	cmd.PersistentFlags().IntVarP(&c.VhostHTTPSPort, "vhost_https_port", "", 0, "vhost https port")
	cmd.PersistentFlags().Int64VarP(&c.VhostHTTPTimeout, "vhost_http_timeout", "", 60, "vhost http response header timeout")
	cmd.PersistentFlags().StringVarP(&c.WebServer.Addr, "dashboard_addr", "", "0.0.0.0", "dashboard address")
	cmd.PersistentFlags().IntVarP(&c.WebServer.Port, "dashboard_port", "", 0, "dashboard port")
	cmd.PersistentFlags().StringVarP(&c.WebServer.User, "dashboard_user", "", "admin", "dashboard user")
	cmd.PersistentFlags().StringVarP(&c.WebServer.Password, "dashboard_pwd", "", "admin", "dashboard password")
	cmd.PersistentFlags().BoolVarP(&c.EnablePrometheus, "enable_prometheus", "", false, "enable prometheus dashboard")
	cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "log file")
	cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "log level")
	cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "log max days")
	cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "disable log color in console")
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "auth token")
	cmd.PersistentFlags().StringVarP(&c.SubDomainHost, "subdomain_host", "", "", "subdomain host")
	cmd.PersistentFlags().VarP(&PortsRangeSliceFlag{V: &c.AllowPorts}, "allow_ports", "", "allow ports")
	cmd.PersistentFlags().Int64VarP(&c.MaxPortsPerClient, "max_ports_per_client", "", 0, "max ports per client")
	cmd.PersistentFlags().BoolVarP(&c.Transport.TLS.Force, "tls_only", "", false, "frps tls only")

	webServerTLS := v1.TLSConfig{}
	cmd.PersistentFlags().StringVarP(&webServerTLS.CertFile, "dashboard_tls_cert_file", "", "", "dashboard tls cert file")
	cmd.PersistentFlags().StringVarP(&webServerTLS.KeyFile, "dashboard_tls_key_file", "", "", "dashboard tls key file")
	cmd.PersistentFlags().VarP(&BoolFuncFlag{
		TrueFunc: func() { c.WebServer.TLS = &webServerTLS },
	}, "dashboard_tls_mode", "", "if enable dashboard tls mode")
}
