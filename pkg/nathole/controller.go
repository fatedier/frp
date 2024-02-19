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

package nathole

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fatedier/golib/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/transport"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/util"
)

// NatHoleTimeout seconds.
var NatHoleTimeout int64 = 10

func NewTransactionID() string {
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", time.Now().Unix(), id)
}

type ClientCfg struct {
	name       string
	sk         string
	allowUsers []string
	sidCh      chan string
}

type Session struct {
	sid            string
	analysisKey    string
	recommandMode  int
	recommandIndex int

	visitorMsg         *msg.NatHoleVisitor
	visitorTransporter transport.MessageTransporter
	vResp              *msg.NatHoleResp
	vNatFeature        *NatFeature
	vBehavior          RecommandBehavior

	clientMsg         *msg.NatHoleClient
	clientTransporter transport.MessageTransporter
	cResp             *msg.NatHoleResp
	cNatFeature       *NatFeature
	cBehavior         RecommandBehavior

	notifyCh chan struct{}
}

func (s *Session) genAnalysisKey() {
	hash := md5.New()
	vIPs := lo.Uniq(parseIPs(s.visitorMsg.MappedAddrs))
	if len(vIPs) > 0 {
		hash.Write([]byte(vIPs[0]))
	}
	hash.Write([]byte(s.vNatFeature.NatType))
	hash.Write([]byte(s.vNatFeature.Behavior))
	hash.Write([]byte(strconv.FormatBool(s.vNatFeature.RegularPortsChange)))

	cIPs := lo.Uniq(parseIPs(s.clientMsg.MappedAddrs))
	if len(cIPs) > 0 {
		hash.Write([]byte(cIPs[0]))
	}
	hash.Write([]byte(s.cNatFeature.NatType))
	hash.Write([]byte(s.cNatFeature.Behavior))
	hash.Write([]byte(strconv.FormatBool(s.cNatFeature.RegularPortsChange)))
	s.analysisKey = hex.EncodeToString(hash.Sum(nil))
}

type Controller struct {
	clientCfgs map[string]*ClientCfg
	sessions   map[string]*Session
	analyzer   *Analyzer

	mu sync.RWMutex
}

func NewController(analysisDataReserveDuration time.Duration) (*Controller, error) {
	return &Controller{
		clientCfgs: make(map[string]*ClientCfg),
		sessions:   make(map[string]*Session),
		analyzer:   NewAnalyzer(analysisDataReserveDuration),
	}, nil
}

func (c *Controller) CleanWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			count, total := c.analyzer.Clean()
			log.Trace("clean %d/%d nathole analysis data, cost %v", count, total, time.Since(start))
		case <-ctx.Done():
			return
		}
	}
}

func (c *Controller) ListenClient(name string, sk string, allowUsers []string) (chan string, error) {
	cfg := &ClientCfg{
		name:       name,
		sk:         sk,
		allowUsers: allowUsers,
		sidCh:      make(chan string),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.clientCfgs[name]; ok {
		return nil, fmt.Errorf("proxy [%s] is repeated", name)
	}
	c.clientCfgs[name] = cfg
	return cfg.sidCh, nil
}

func (c *Controller) CloseClient(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.clientCfgs, name)
}

func (c *Controller) GenSid() string {
	t := time.Now().Unix()
	id, _ := util.RandID()
	return fmt.Sprintf("%d%s", t, id)
}

func (c *Controller) HandleVisitor(m *msg.NatHoleVisitor, transporter transport.MessageTransporter, visitorUser string) {
	if m.PreCheck {
		cfg, ok := c.clientCfgs[m.ProxyName]
		if !ok {
			_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, fmt.Sprintf("xtcp server for [%s] doesn't exist", m.ProxyName)))
			return
		}
		if !lo.Contains(cfg.allowUsers, visitorUser) && !lo.Contains(cfg.allowUsers, "*") {
			_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, fmt.Sprintf("xtcp visitor user [%s] not allowed for [%s]", visitorUser, m.ProxyName)))
			return
		}
		_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, ""))
		return
	}

	sid := c.GenSid()
	session := &Session{
		sid:                sid,
		visitorMsg:         m,
		visitorTransporter: transporter,
		notifyCh:           make(chan struct{}, 1),
	}
	var (
		clientCfg *ClientCfg
		ok        bool
	)
	err := func() error {
		c.mu.Lock()
		defer c.mu.Unlock()

		clientCfg, ok = c.clientCfgs[m.ProxyName]
		if !ok {
			return fmt.Errorf("xtcp server for [%s] doesn't exist", m.ProxyName)
		}
		if !util.ConstantTimeEqString(m.SignKey, util.GetAuthKey(clientCfg.sk, m.Timestamp)) {
			return fmt.Errorf("xtcp connection of [%s] auth failed", m.ProxyName)
		}
		c.sessions[sid] = session
		return nil
	}()
	if err != nil {
		log.Warn("handle visitorMsg error: %v", err)
		_ = transporter.Send(c.GenNatHoleResponse(m.TransactionID, nil, err.Error()))
		return
	}
	log.Trace("handle visitor message, sid [%s], server name: %s", sid, m.ProxyName)

	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.sessions, sid)
	}()

	if err := errors.PanicToError(func() {
		clientCfg.sidCh <- sid
	}); err != nil {
		return
	}

	// wait for NatHoleClient message
	select {
	case <-session.notifyCh:
	case <-time.After(time.Duration(NatHoleTimeout) * time.Second):
		log.Debug("wait for NatHoleClient message timeout, sid [%s]", sid)
		return
	}

	// Make hole-punching decisions based on the NAT information of the client and visitor.
	vResp, cResp, err := c.analysis(session)
	if err != nil {
		log.Debug("sid [%s] analysis error: %v", err)
		vResp = c.GenNatHoleResponse(session.visitorMsg.TransactionID, nil, err.Error())
		cResp = c.GenNatHoleResponse(session.clientMsg.TransactionID, nil, err.Error())
	}
	session.cResp = cResp
	session.vResp = vResp

	// send response to visitor and client
	var g errgroup.Group
	g.Go(func() error {
		// if it's sender, wait for a while to make sure the client has send the detect messages
		if vResp.DetectBehavior.Role == "sender" {
			time.Sleep(1 * time.Second)
		}
		_ = session.visitorTransporter.Send(vResp)
		return nil
	})
	g.Go(func() error {
		// if it's sender, wait for a while to make sure the client has send the detect messages
		if cResp.DetectBehavior.Role == "sender" {
			time.Sleep(1 * time.Second)
		}
		_ = session.clientTransporter.Send(cResp)
		return nil
	})
	_ = g.Wait()

	time.Sleep(time.Duration(cResp.DetectBehavior.ReadTimeoutMs+30000) * time.Millisecond)
}

func (c *Controller) HandleClient(m *msg.NatHoleClient, transporter transport.MessageTransporter) {
	c.mu.RLock()
	session, ok := c.sessions[m.Sid]
	c.mu.RUnlock()
	if !ok {
		return
	}
	log.Trace("handle client message, sid [%s], server name: %s", session.sid, m.ProxyName)
	session.clientMsg = m
	session.clientTransporter = transporter
	select {
	case session.notifyCh <- struct{}{}:
	default:
	}
}

func (c *Controller) HandleReport(m *msg.NatHoleReport) {
	c.mu.RLock()
	session, ok := c.sessions[m.Sid]
	c.mu.RUnlock()
	if !ok {
		log.Trace("sid [%s] report make hole success: %v, but session not found", m.Sid, m.Success)
		return
	}
	if m.Success {
		c.analyzer.ReportSuccess(session.analysisKey, session.recommandMode, session.recommandIndex)
	}
	log.Info("sid [%s] report make hole success: %v, mode %v, index %v",
		m.Sid, m.Success, session.recommandMode, session.recommandIndex)
}

func (c *Controller) GenNatHoleResponse(transactionID string, session *Session, errInfo string) *msg.NatHoleResp {
	var sid string
	if session != nil {
		sid = session.sid
	}
	return &msg.NatHoleResp{
		TransactionID: transactionID,
		Sid:           sid,
		Error:         errInfo,
	}
}

// analysis analyzes the NAT type and behavior of the visitor and client, then makes hole-punching decisions.
// return the response to the visitor and client.
func (c *Controller) analysis(session *Session) (*msg.NatHoleResp, *msg.NatHoleResp, error) {
	cm := session.clientMsg
	vm := session.visitorMsg

	cNatFeature, err := ClassifyNATFeature(cm.MappedAddrs, parseIPs(cm.AssistedAddrs))
	if err != nil {
		return nil, nil, fmt.Errorf("classify client nat feature error: %v", err)
	}

	vNatFeature, err := ClassifyNATFeature(vm.MappedAddrs, parseIPs(vm.AssistedAddrs))
	if err != nil {
		return nil, nil, fmt.Errorf("classify visitor nat feature error: %v", err)
	}
	session.cNatFeature = cNatFeature
	session.vNatFeature = vNatFeature
	session.genAnalysisKey()

	mode, index, cBehavior, vBehavior := c.analyzer.GetRecommandBehaviors(session.analysisKey, cNatFeature, vNatFeature)
	session.recommandMode = mode
	session.recommandIndex = index
	session.cBehavior = cBehavior
	session.vBehavior = vBehavior

	timeoutMs := max(cBehavior.SendDelayMs, vBehavior.SendDelayMs) + 5000
	if cBehavior.ListenRandomPorts > 0 || vBehavior.ListenRandomPorts > 0 {
		timeoutMs += 30000
	}

	protocol := vm.Protocol
	vResp := &msg.NatHoleResp{
		TransactionID:  vm.TransactionID,
		Sid:            session.sid,
		Protocol:       protocol,
		CandidateAddrs: lo.Uniq(cm.MappedAddrs),
		AssistedAddrs:  lo.Uniq(cm.AssistedAddrs),
		DetectBehavior: msg.NatHoleDetectBehavior{
			Mode:              mode,
			Role:              vBehavior.Role,
			TTL:               vBehavior.TTL,
			SendDelayMs:       vBehavior.SendDelayMs,
			ReadTimeoutMs:     timeoutMs - vBehavior.SendDelayMs,
			SendRandomPorts:   vBehavior.PortsRandomNumber,
			ListenRandomPorts: vBehavior.ListenRandomPorts,
			CandidatePorts:    getRangePorts(cm.MappedAddrs, cNatFeature.PortsDifference, vBehavior.PortsRangeNumber),
		},
	}
	cResp := &msg.NatHoleResp{
		TransactionID:  cm.TransactionID,
		Sid:            session.sid,
		Protocol:       protocol,
		CandidateAddrs: lo.Uniq(vm.MappedAddrs),
		AssistedAddrs:  lo.Uniq(vm.AssistedAddrs),
		DetectBehavior: msg.NatHoleDetectBehavior{
			Mode:              mode,
			Role:              cBehavior.Role,
			TTL:               cBehavior.TTL,
			SendDelayMs:       cBehavior.SendDelayMs,
			ReadTimeoutMs:     timeoutMs - cBehavior.SendDelayMs,
			SendRandomPorts:   cBehavior.PortsRandomNumber,
			ListenRandomPorts: cBehavior.ListenRandomPorts,
			CandidatePorts:    getRangePorts(vm.MappedAddrs, vNatFeature.PortsDifference, cBehavior.PortsRangeNumber),
		},
	}

	log.Debug("sid [%s] visitor nat: %+v, candidateAddrs: %v; client nat: %+v, candidateAddrs: %v, protocol: %s",
		session.sid, *vNatFeature, vm.MappedAddrs, *cNatFeature, cm.MappedAddrs, protocol)
	log.Debug("sid [%s] visitor detect behavior: %+v", session.sid, vResp.DetectBehavior)
	log.Debug("sid [%s] client detect behavior: %+v", session.sid, cResp.DetectBehavior)
	return vResp, cResp, nil
}

func getRangePorts(addrs []string, difference, maxNumber int) []msg.PortsRange {
	if maxNumber <= 0 {
		return nil
	}

	addr, err := lo.Last(addrs)
	if err != nil {
		return nil
	}
	var ports []msg.PortsRange
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil
	}
	ports = append(ports, msg.PortsRange{
		From: max(port-difference-5, port-maxNumber, 1),
		To:   min(port+difference+5, port+maxNumber, 65535),
	})
	return ports
}
