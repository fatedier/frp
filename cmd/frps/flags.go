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

package main

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config/types"
	v1 "github.com/fatedier/frp/pkg/config/v1"
)

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

func RegisterServerConfigFlags(cmd *cobra.Command, c *v1.ServerConfig) {
	cmd.PersistentFlags().StringVarP(&c.BindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	cmd.PersistentFlags().IntVarP(&c.BindPort, "bind_port", "p", 7000, "bind port")
	cmd.PersistentFlags().IntVarP(&c.KCPBindPort, "kcp_bind_port", "", 0, "kcp bind udp port")
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
