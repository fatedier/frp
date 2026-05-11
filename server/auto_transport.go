// Copyright 2026 The frp Authors
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

package server

import (
	"encoding/json"
	"fmt"
	"net"
	"slices"

	"github.com/samber/lo"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	splugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server/metrics"
)

type autoTransportEntry struct {
	Protocols []string
	Port      int
}

func (svr *Service) autoTransportEnabled() bool {
	return svr.cfg.Transport.Protocol == v1.TransportProtocolAuto && lo.FromPtr(svr.cfg.Transport.Auto.Enabled)
}

func (svr *Service) autoTransportEndpoints() []msg.TransportEndpoint {
	cfg := svr.cfg
	advertise := cfg.Transport.Auto.AdvertiseProtocols
	has := func(protocol string) bool {
		return slices.Contains(advertise, protocol)
	}

	endpoints := make([]msg.TransportEndpoint, 0, 5)
	add := func(protocol string, port int) {
		if port <= 0 || !has(protocol) {
			return
		}
		endpoints = append(endpoints, msg.TransportEndpoint{
			Protocol: protocol,
			Addr:     cfg.BindAddr,
			Port:     port,
			Enabled:  true,
		})
	}

	add(v1.TransportProtocolTCP, cfg.BindPort)
	add(v1.TransportProtocolKCP, cfg.KCPBindPort)
	add(v1.TransportProtocolQUIC, cfg.QUICBindPort)
	add(v1.TransportProtocolWebsocket, cfg.BindPort)
	add(v1.TransportProtocolWSS, cfg.BindPort)
	return endpoints
}

func (svr *Service) verifyAutoLogin(conn net.Conn, login *msg.Login) error {
	if svr.pluginManager != nil {
		content := &splugin.LoginContent{
			Login:         *login,
			ClientAddress: conn.RemoteAddr().String(),
		}
		retContent, err := svr.pluginManager.Login(content)
		if err != nil {
			return err
		}
		login = &retContent.Login
	}
	return svr.auth.Verifier.VerifyLogin(login)
}

func autoLoginFromClientHello(m *msg.ClientHelloAuto) *msg.Login {
	return autoLoginWithFallback(m.Login, m.PrivilegeKey, m.Timestamp)
}

func autoLoginFromProbe(m *msg.ProbeTransport) *msg.Login {
	return autoLoginWithFallback(m.Login, m.PrivilegeKey, m.Timestamp)
}

func autoLoginWithFallback(login *msg.Login, privilegeKey string, timestamp int64) *msg.Login {
	if login == nil {
		return &msg.Login{
			PrivilegeKey: privilegeKey,
			Timestamp:    timestamp,
		}
	}
	out := *login
	if out.PrivilegeKey == "" {
		out.PrivilegeKey = privilegeKey
	}
	if out.Timestamp == 0 {
		out.Timestamp = timestamp
	}
	return &out
}

func (svr *Service) handleClientHelloAuto(conn net.Conn, m *msg.ClientHelloAuto) {
	resp := &msg.ServerHelloAuto{
		ProtocolMode:       svr.cfg.Transport.Protocol,
		AutoEnabled:        svr.autoTransportEnabled(),
		AllowDynamicSwitch: lo.FromPtr(svr.cfg.Transport.Auto.AllowDynamicSwitch),
		PreferOrder:        append([]string(nil), svr.cfg.Transport.Auto.PreferOrder...),
		Transports:         svr.autoTransportEndpoints(),
		ServerAutoVersion:  msg.AutoTransportVersion,
	}
	if err := validateClientAutoVersion(m.ClientAutoVersion); err != nil {
		resp.Error = err.Error()
		resp.AutoEnabled = false
	} else if err := svr.verifyAutoLogin(conn, autoLoginFromClientHello(m)); err != nil {
		resp.Error = fmt.Sprintf("auto transport auth failed: %v", err)
		resp.AutoEnabled = false
	}
	metrics.Server.AutoNegotiation(resp.Error == "" && resp.AutoEnabled)
	_ = msg.WriteMsg(conn, resp)
	_ = conn.Close()
}

func (svr *Service) handleProbeTransport(conn net.Conn, m *msg.ProbeTransport, entry autoTransportEntry) {
	resp := &msg.ProbeTransportResp{
		Protocol:          m.Protocol,
		Port:              m.Port,
		ServerAutoVersion: msg.AutoTransportVersion,
	}
	if err := validateClientAutoVersion(m.ClientAutoVersion); err != nil {
		resp.Error = err.Error()
	} else if err := svr.verifyAutoLogin(conn, autoLoginFromProbe(m)); err != nil {
		resp.Error = fmt.Sprintf("auto transport probe auth failed: %v", err)
	} else if err := svr.validateSelectedTransportForEntry(m.Protocol, m.Addr, m.Port, entry); err != nil {
		resp.Error = err.Error()
	}
	_ = msg.WriteMsg(conn, resp)
	_ = conn.Close()
}

func validateClientAutoVersion(version uint32) error {
	if version != msg.AutoTransportVersion {
		return fmt.Errorf("unsupported client auto transport version %d", version)
	}
	return nil
}

func (svr *Service) validateSelectedTransport(protocol string, addr string, port int) error {
	if !svr.autoTransportEnabled() {
		return fmt.Errorf("server auto transport is not enabled")
	}
	for _, ep := range svr.autoTransportEndpoints() {
		if ep.Enabled && ep.Protocol == protocol && ep.Port == port && advertisedAddrMatches(ep.Addr, addr) {
			return nil
		}
	}
	return fmt.Errorf("transport %s@%s:%d was not advertised by server", protocol, addr, port)
}

func (svr *Service) validateSelectedTransportForEntry(protocol string, addr string, port int, entry autoTransportEntry) error {
	if err := svr.validateSelectedTransport(protocol, addr, port); err != nil {
		return err
	}
	if len(entry.Protocols) == 0 {
		return nil
	}
	if !slices.Contains(entry.Protocols, protocol) || (entry.Port > 0 && entry.Port != port) {
		return fmt.Errorf("transport %s@%s:%d does not match connection entry %v@%d",
			protocol, addr, port, entry.Protocols, entry.Port)
	}
	return nil
}

func advertisedAddrMatches(advertised string, selected string) bool {
	if advertised == "" || advertised == "0.0.0.0" || advertised == "::" || advertised == "[::]" {
		return true
	}
	return advertised == selected
}

func (svr *Service) logSelectedTransport(
	runID string,
	protocol string,
	port int,
	reason string,
	scores map[string]int64,
) {
	if protocol == "" {
		return
	}
	event := map[string]any{
		"event":    "transport_selected",
		"run_id":   runID,
		"protocol": protocol,
		"port":     port,
		"reason":   reason,
		"scores":   scores,
	}
	if data, err := json.Marshal(event); err == nil {
		log.Infof("auto transport event: %s", string(data))
	}
	log.Infof("auto transport selected: run_id [%s] protocol [%s] port [%d] reason [%s]",
		runID, protocol, port, reason)
}

func (svr *Service) logTransportSwitch(
	runID string,
	oldProtocol string,
	newProtocol string,
	reason string,
	scores map[string]int64,
) {
	if oldProtocol == "" || newProtocol == "" || oldProtocol == newProtocol {
		return
	}
	event := map[string]any{
		"event":        "transport_switch",
		"run_id":       runID,
		"old_protocol": oldProtocol,
		"new_protocol": newProtocol,
		"reason":       reason,
		"scores":       scores,
	}
	if data, err := json.Marshal(event); err == nil {
		log.Infof("auto transport event: %s", string(data))
	}
	log.Infof("auto transport switch: run_id [%s] old_protocol [%s] new_protocol [%s] reason [%s]",
		runID, oldProtocol, newProtocol, reason)
}

func (svr *Service) logRejectedTransport(protocol string, port int, err error) {
	event := map[string]any{
		"event":    "transport_reject",
		"protocol": protocol,
		"port":     port,
		"reason":   err.Error(),
	}
	if data, jsonErr := json.Marshal(event); jsonErr == nil {
		log.Warnf("auto transport event: %s", string(data))
	}
}
