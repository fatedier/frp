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

package sub

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
)

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

func RegisterProxyFlags(cmd *cobra.Command, c v1.ProxyConfigurer) {
	registerProxyBaseConfigFlags(cmd, c.GetBaseConfig())
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
	case *v1.STCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
	case *v1.SUDPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
	case *v1.XTCPProxyConfig:
		cmd.Flags().StringVarP(&cc.Secretkey, "sk", "", "", "secret key")
	}
}

func registerProxyBaseConfigFlags(cmd *cobra.Command, c *v1.ProxyBaseConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "proxy_name", "n", "", "proxy name")
	cmd.Flags().StringVarP(&c.LocalIP, "local_ip", "i", "127.0.0.1", "local ip")
	cmd.Flags().IntVarP(&c.LocalPort, "local_port", "l", 0, "local port")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
	cmd.Flags().StringVarP(&c.Transport.BandwidthLimitMode, "bandwidth_limit_mode", "", types.BandwidthLimitModeClient, "bandwidth limit mode")
	cmd.Flags().VarP(&BandwidthQuantityFlag{V: &c.Transport.BandwidthLimit}, "bandwidth_limit", "", "bandwidth limit (e.g. 100KB or 1MB)")
}

func registerProxyDomainConfigFlags(cmd *cobra.Command, c *v1.DomainConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringSliceVarP(&c.CustomDomains, "custom_domain", "d", []string{}, "custom domains")
	cmd.Flags().StringVarP(&c.SubDomain, "sd", "", "", "sub domain")
}

func RegisterVisitorFlags(cmd *cobra.Command, c v1.VisitorConfigurer) {
	registerVisitorBaseConfigFlags(cmd, c.GetBaseConfig())

	// add visitor flags if exist
}

func registerVisitorBaseConfigFlags(cmd *cobra.Command, c *v1.VisitorBaseConfig) {
	if c == nil {
		return
	}
	cmd.Flags().StringVarP(&c.Name, "visitor_name", "n", "", "visitor name")
	cmd.Flags().BoolVarP(&c.Transport.UseEncryption, "ue", "", false, "use encryption")
	cmd.Flags().BoolVarP(&c.Transport.UseCompression, "uc", "", false, "use compression")
	cmd.Flags().StringVarP(&c.SecretKey, "sk", "", "", "secret key")
	cmd.Flags().StringVarP(&c.ServerName, "server_name", "", "", "server name")
	cmd.Flags().StringVarP(&c.BindAddr, "bind_addr", "", "", "bind addr")
	cmd.Flags().IntVarP(&c.BindPort, "bind_port", "", 0, "bind port")
}

func RegisterClientCommonConfigFlags(cmd *cobra.Command, c *v1.ClientCommonConfig) {
	cmd.PersistentFlags().StringVarP(&c.ServerAddr, "server_addr", "s", "127.0.0.1", "frp server's address")
	cmd.PersistentFlags().IntVarP(&c.ServerPort, "server_port", "P", 7000, "frp server's port")
	cmd.PersistentFlags().StringVarP(&c.User, "user", "u", "", "user")
	cmd.PersistentFlags().StringVarP(&c.Transport.Protocol, "protocol", "p", "tcp",
		fmt.Sprintf("optional values are %v", validation.SupportedTransportProtocols))
	cmd.PersistentFlags().StringVarP(&c.Auth.Token, "token", "t", "", "auth token")
	cmd.PersistentFlags().StringVarP(&c.Log.Level, "log_level", "", "info", "log level")
	cmd.PersistentFlags().StringVarP(&c.Log.To, "log_file", "", "console", "console or file path")
	cmd.PersistentFlags().Int64VarP(&c.Log.MaxDays, "log_max_days", "", 3, "log file reversed days")
	cmd.PersistentFlags().BoolVarP(&c.Log.DisablePrintColor, "disable_log_color", "", false, "disable log color in console")
	cmd.PersistentFlags().StringVarP(&c.Transport.TLS.ServerName, "tls_server_name", "", "", "specify the custom server name of tls certificate")
	cmd.PersistentFlags().StringVarP(&c.DNSServer, "dns_server", "", "", "specify dns server instead of using system default one")

	c.Transport.TLS.Enable = cmd.PersistentFlags().BoolP("tls_enable", "", true, "enable frpc tls")
}
