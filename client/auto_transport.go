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

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/fatedier/frp/pkg/auth"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/log"
)

const (
	autoTransportStateBootstrap      = "BOOTSTRAP"
	autoTransportStateNegotiating    = "NEGOTIATING"
	autoTransportStateProbing        = "PROBING"
	autoTransportStateSelected       = "SELECTED"
	autoTransportStateConnected      = "CONNECTED"
	autoTransportStateDegraded       = "DEGRADED"
	autoTransportStateSwitching      = "SWITCHING"
	autoTransportStateBackoff        = "BACKOFF"
	autoTransportStateFallbackStatic = "FALLBACK_STATIC"

	autoTransportReasonStartup       = "startup"
	autoTransportReasonControlClosed = "control_closed"
	autoTransportReasonLoginFailure  = "login_failure"
	autoTransportReasonHeartbeatRTT  = "heartbeat_rtt_degraded"
	autoTransportReasonHeartbeat     = "heartbeat_timeout"
	autoTransportReasonWorkConn      = "work_conn_failure"
)

const autoTransportHysteresisRatio = 1.2

type autoTransportCandidate struct {
	Protocol string
	Addr     string
	Port     int
	Priority int
	Source   string
}

type autoTransportSelection struct {
	Cfg        *v1.ClientCommonConfig
	Candidate  autoTransportCandidate
	SendSelect bool
	Dynamic    bool
	Reason     string
	Scores     map[string]int64
}

type autoTransportProbeResult struct {
	Candidate   autoTransportCandidate
	Successes   int
	AvgRTT      time.Duration
	Score       int64
	ScoreDetail AutoTransportScoreDetail
	Err         error
}

type autoTransportManager struct {
	common           *v1.ClientCommonConfig
	auth             *auth.ClientAuth
	connectorCreator func(context.Context, *v1.ClientCommonConfig) Connector
	statePath        string

	mu               sync.Mutex
	selected         *autoTransportCandidate
	previousProtocol string
	selectedAt       time.Time
	selectedScore    int64
	lastGood         string
	state            string
	lastReason       string
	lastError        string
	lastScores       map[string]int64
	lastScoreDetails map[string]AutoTransportScoreDetail
	lastSuccessRates map[string]float64
	lastProbeRTTMs   map[string]int64
	lastProbeErrors  map[string]string
	lastSwitchAt     time.Time
	switchCount      int64
	lastDynamic      bool
	failures         map[string]int
	blacklistUntil   map[string]time.Time

	lastHeartbeatRTT       time.Duration
	avgHeartbeatRTT        time.Duration
	heartbeatTimeouts      int
	workConnFailures       int
	qualityDegradeCount    int
	degradeEvents          int64
	runtimeFailureReported bool
}

type AutoTransportStatus struct {
	AutoEnabled          bool                                `json:"autoEnabled"`
	State                string                              `json:"state"`
	CurrentProtocol      string                              `json:"currentProtocol,omitempty"`
	CurrentAddr          string                              `json:"currentAddr,omitempty"`
	CurrentPort          int                                 `json:"currentPort,omitempty"`
	CurrentScore         int64                               `json:"currentScore,omitempty"`
	PreviousProtocol     string                              `json:"previousProtocol,omitempty"`
	LastGoodProtocol     string                              `json:"lastGoodProtocol,omitempty"`
	LastSwitchReason     string                              `json:"lastSwitchReason,omitempty"`
	LastError            string                              `json:"lastError,omitempty"`
	SwitchCount          int64                               `json:"switchCount"`
	Dynamic              bool                                `json:"dynamic"`
	StickyRemainingSec   int64                               `json:"stickyRemainingSec,omitempty"`
	CooldownRemainingSec int64                               `json:"cooldownRemainingSec,omitempty"`
	BlacklistProtocols   []string                            `json:"blacklistProtocols,omitempty"`
	Strategy             string                              `json:"strategy,omitempty"`
	LastScores           map[string]int64                    `json:"lastScores,omitempty"`
	LastScoreDetails     map[string]AutoTransportScoreDetail `json:"lastScoreDetails,omitempty"`
	LastSuccessRates     map[string]float64                  `json:"lastSuccessRates,omitempty"`
	LastProbeRTTMs       map[string]int64                    `json:"lastProbeRTTMs,omitempty"`
	LastProbeErrors      map[string]string                   `json:"lastProbeErrors,omitempty"`
	HeartbeatRTTMs       int64                               `json:"heartbeatRTTMs,omitempty"`
	AvgHeartbeatRTTMs    int64                               `json:"avgHeartbeatRTTMs,omitempty"`
	HeartbeatTimeouts    int                                 `json:"heartbeatTimeouts,omitempty"`
	WorkConnFailures     int                                 `json:"workConnFailures,omitempty"`
	QualityDegradeCount  int                                 `json:"qualityDegradeCount,omitempty"`
	DegradeEvents        int64                               `json:"degradeEvents,omitempty"`
	PersistLastGood      bool                                `json:"persistLastGood"`
}

type autoTransportPersistState struct {
	LastGoodProtocol string `json:"lastGoodProtocol,omitempty"`
}

func newAutoTransportManager(
	common *v1.ClientCommonConfig,
	authRuntime *auth.ClientAuth,
	connectorCreator func(context.Context, *v1.ClientCommonConfig) Connector,
	statePaths ...string,
) *autoTransportManager {
	m := &autoTransportManager{
		common:           common,
		auth:             authRuntime,
		connectorCreator: connectorCreator,
		lastScores:       make(map[string]int64),
		lastScoreDetails: make(map[string]AutoTransportScoreDetail),
		lastSuccessRates: make(map[string]float64),
		lastProbeRTTMs:   make(map[string]int64),
		lastProbeErrors:  make(map[string]string),
		failures:         make(map[string]int),
		blacklistUntil:   make(map[string]time.Time),
	}
	if len(statePaths) > 0 {
		m.statePath = statePaths[0]
	}
	m.loadLastGood()
	return m
}

func (m *autoTransportManager) selectTransport(ctx context.Context, reason string) (*autoTransportSelection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = autoTransportStateBootstrap
	m.lastReason = reason
	m.lastError = ""
	log.Infof("auto transport: start selection, reason [%s]", reason)
	serverHello, err := m.bootstrap(ctx)
	if err != nil {
		m.lastError = err.Error()
		log.Warnf("auto transport: bootstrap failed, fallback to static tcp: %v", err)
		return m.staticFallbackSelection(), nil
	}
	if serverHello.Error != "" {
		m.lastError = serverHello.Error
		return nil, errors.New(serverHello.Error)
	}
	if serverHello.ServerAutoVersion != msg.AutoTransportVersion {
		m.lastError = fmt.Sprintf("unsupported server auto transport version %d", serverHello.ServerAutoVersion)
		log.Warnf("auto transport: %s, fallback to static tcp", m.lastError)
		return m.staticFallbackSelection(), nil
	}
	m.state = autoTransportStateNegotiating
	if serverHello.ProtocolMode != v1.TransportProtocolAuto || !serverHello.AutoEnabled {
		log.Infof("auto transport: server mode [%s] autoEnabled [%v], fallback to static tcp",
			serverHello.ProtocolMode, serverHello.AutoEnabled)
		return m.staticFallbackSelection(), nil
	}

	candidates := m.buildCandidates(serverHello)
	if len(candidates) == 0 {
		m.lastError = "no common transport candidates"
		return nil, fmt.Errorf("auto transport: no common transport candidates")
	}
	dynamic := serverHello.AllowDynamicSwitch && len(candidates) >= 2
	if !dynamic {
		candidates = m.fixedModeCandidatesLocked(candidates)
	}

	m.state = autoTransportStateProbing
	selectedResult, results, err := m.probeAndSelect(ctx, candidates, reason)
	m.lastScores = scoresFromProbeResults(results)
	m.lastScoreDetails = scoreDetailsFromProbeResults(results)
	m.lastSuccessRates, m.lastProbeRTTMs, m.lastProbeErrors = probeStatusFromResults(
		results,
		m.common.Transport.Auto.ProbeCount,
	)
	if err != nil {
		m.state = autoTransportStateBackoff
		m.lastError = err.Error()
		return nil, err
	}
	for _, result := range results {
		if result.Err != nil {
			log.Infof("auto transport: probe %s@%s:%d failed: %v",
				result.Candidate.Protocol, result.Candidate.Addr, result.Candidate.Port, result.Err)
			continue
		}
		log.Infof("auto transport: probe %s@%s:%d success count [%d] avg_rtt [%s] score [%d]",
			result.Candidate.Protocol, result.Candidate.Addr, result.Candidate.Port,
			result.Successes, result.AvgRTT, result.Score)
	}

	selected := selectedResult.Candidate
	m.recordSelectionLocked(selected, selectedResult.Score, dynamic, reason)
	m.state = autoTransportStateSelected
	m.runtimeFailureReported = false
	cfg := m.configForCandidate(selected)
	log.Infof("auto transport: selected %s@%s:%d dynamic [%v]",
		selected.Protocol, selected.Addr, selected.Port, dynamic)
	return &autoTransportSelection{
		Cfg:        cfg,
		Candidate:  selected,
		SendSelect: true,
		Dynamic:    dynamic,
		Reason:     reason,
		Scores:     copyInt64Map(m.lastScores),
	}, nil
}

func (m *autoTransportManager) reportLoginSuccess(protocol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if protocol == "" {
		return
	}
	m.lastGood = protocol
	m.state = autoTransportStateConnected
	m.lastError = ""
	m.workConnFailures = 0
	m.qualityDegradeCount = 0
	m.runtimeFailureReported = false
	if m.selected != nil && m.selected.Protocol == protocol {
		m.selectedAt = time.Now()
	}
	m.failures[protocol] = 0
	delete(m.blacklistUntil, protocol)
	m.persistLastGood(protocol)
}

func (m *autoTransportManager) reportLoginFailure(protocol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if protocol == "" {
		return
	}
	m.lastReason = autoTransportReasonLoginFailure
	m.lastError = fmt.Sprintf("login failed on %s", protocol)
	if m.selected != nil && m.selected.Protocol == protocol {
		m.selectedAt = time.Time{}
	}
	m.failures[protocol]++
	m.state = autoTransportStateBackoff
	if m.failures[protocol] >= m.common.Transport.Auto.FailureThreshold {
		m.blacklistUntil[protocol] = time.Now().Add(time.Duration(m.common.Transport.Auto.CooldownSec) * time.Second)
		log.Warnf("auto transport: protocol [%s] enters cooldown for %d seconds",
			protocol, m.common.Transport.Auto.CooldownSec)
	}
}

func (m *autoTransportManager) reportHeartbeatRTT(rtt time.Duration) bool {
	if rtt <= 0 {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastHeartbeatRTT = rtt
	m.heartbeatTimeouts = 0
	if m.avgHeartbeatRTT <= 0 {
		m.avgHeartbeatRTT = rtt
		return false
	}

	previousAvg := m.avgHeartbeatRTT
	m.avgHeartbeatRTT = (m.avgHeartbeatRTT*4 + rtt) / 5
	if rtt >= previousAvg*3 && rtt >= time.Second {
		return m.recordDegradeLocked(autoTransportReasonHeartbeatRTT, &m.qualityDegradeCount)
	}
	m.qualityDegradeCount = 0
	if m.state == autoTransportStateDegraded && m.heartbeatTimeouts == 0 && m.workConnFailures == 0 {
		m.state = autoTransportStateConnected
	}
	return false
}

func (m *autoTransportManager) reportHeartbeatTimeout() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.recordDegradeLocked(autoTransportReasonHeartbeat, &m.heartbeatTimeouts)
}

func (m *autoTransportManager) reportWorkConnSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workConnFailures = 0
	if m.state == autoTransportStateDegraded && m.heartbeatTimeouts == 0 && m.qualityDegradeCount == 0 {
		m.state = autoTransportStateConnected
	}
}

func (m *autoTransportManager) reportWorkConnFailure(reason string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if reason == "" {
		reason = autoTransportReasonWorkConn
	}
	return m.recordDegradeLocked(reason, &m.workConnFailures)
}

func (m *autoTransportManager) recordDegradeLocked(reason string, counter *int) bool {
	(*counter)++
	m.degradeEvents++
	m.state = autoTransportStateDegraded
	m.lastReason = reason
	m.lastError = reason

	threshold := m.common.Transport.Auto.DegradeThreshold
	if threshold <= 0 {
		threshold = 1
	}
	if *counter < threshold {
		return false
	}

	m.state = autoTransportStateSwitching
	m.recordRuntimeProtocolFailureLocked(reason)
	return true
}

func (m *autoTransportManager) recordRuntimeProtocolFailureLocked(reason string) {
	if m.runtimeFailureReported || m.selected == nil {
		return
	}

	protocol := m.selected.Protocol
	m.runtimeFailureReported = true
	m.failures[protocol]++
	log.Warnf("auto transport: runtime failure on protocol [%s], reason [%s], count [%d]",
		protocol, reason, m.failures[protocol])
	if m.failures[protocol] >= m.common.Transport.Auto.FailureThreshold {
		m.blacklistUntil[protocol] = time.Now().Add(time.Duration(m.common.Transport.Auto.CooldownSec) * time.Second)
		log.Warnf("auto transport: protocol [%s] enters cooldown for %d seconds",
			protocol, m.common.Transport.Auto.CooldownSec)
	}
}

func (m *autoTransportManager) staticFallbackSelection() *autoTransportSelection {
	cfg := cloneClientCommonConfig(m.common)
	cfg.Transport.Protocol = v1.TransportProtocolTCP
	cfg.ServerPort = m.common.Transport.Auto.BootstrapPort
	candidate := autoTransportCandidate{
		Protocol: v1.TransportProtocolTCP,
		Addr:     cfg.ServerAddr,
		Port:     cfg.ServerPort,
		Source:   "fallback",
	}
	m.recordSelectionLocked(candidate, 0, false, m.lastReason)
	m.state = autoTransportStateFallbackStatic
	return &autoTransportSelection{
		Cfg:       cfg,
		Candidate: candidate,
		Reason:    m.lastReason,
		Scores:    copyInt64Map(m.lastScores),
	}
}

func (m *autoTransportManager) status() AutoTransportStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := AutoTransportStatus{
		AutoEnabled:          true,
		State:                m.state,
		CurrentScore:         m.selectedScore,
		PreviousProtocol:     m.previousProtocol,
		LastGoodProtocol:     m.lastGood,
		LastSwitchReason:     m.lastReason,
		LastError:            m.lastError,
		SwitchCount:          m.switchCount,
		Dynamic:              m.lastDynamic,
		StickyRemainingSec:   m.stickyRemainingSecLocked(),
		CooldownRemainingSec: m.cooldownRemainingSecLocked(),
		BlacklistProtocols:   m.activeBlacklistProtocols(),
		Strategy:             m.common.Transport.Auto.Strategy,
		LastScores:           copyInt64Map(m.lastScores),
		LastScoreDetails:     copyScoreDetailsMap(m.lastScoreDetails),
		LastSuccessRates:     copyFloat64Map(m.lastSuccessRates),
		LastProbeRTTMs:       copyInt64Map(m.lastProbeRTTMs),
		LastProbeErrors:      copyStringMap(m.lastProbeErrors),
		HeartbeatRTTMs:       m.lastHeartbeatRTT.Milliseconds(),
		AvgHeartbeatRTTMs:    m.avgHeartbeatRTT.Milliseconds(),
		HeartbeatTimeouts:    m.heartbeatTimeouts,
		WorkConnFailures:     m.workConnFailures,
		QualityDegradeCount:  m.qualityDegradeCount,
		DegradeEvents:        m.degradeEvents,
		PersistLastGood:      m.persistLastGoodEnabled(),
	}
	if status.State == "" {
		status.State = autoTransportStateBootstrap
	}
	if m.selected != nil {
		status.CurrentProtocol = m.selected.Protocol
		status.CurrentAddr = m.selected.Addr
		status.CurrentPort = m.selected.Port
	}
	return status
}

func (m *autoTransportManager) bootstrap(ctx context.Context) (*msg.ServerHelloAuto, error) {
	cfg := cloneClientCommonConfig(m.common)
	cfg.Transport.Protocol = v1.TransportProtocolTCP
	cfg.ServerPort = m.common.Transport.Auto.BootstrapPort

	timeout := time.Duration(m.common.Transport.Auto.ProbeTimeoutMs) * time.Millisecond
	doCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	connector := m.connectorCreator(doCtx, cfg)
	if err := connector.Open(); err != nil {
		return nil, err
	}
	defer connector.Close()

	conn, err := connector.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	hello := &msg.ClientHelloAuto{
		ProtocolMode:       v1.TransportProtocolAuto,
		ClientCandidates:   m.clientCandidates(),
		AllowUDP:           m.allowUDP(),
		HasProxyURL:        m.common.Transport.ProxyURL != "",
		TLSRequired:        lo.FromPtr(m.common.Transport.TLS.Enable),
		Strategy:           m.common.Transport.Auto.Strategy,
		LastGoodProtocol:   m.lastGood,
		BlacklistProtocols: m.activeBlacklistProtocols(),
		ClientAutoVersion:  msg.AutoTransportVersion,
	}
	if err := m.setAutoAuth(hello); err != nil {
		return nil, err
	}
	if err := msg.WriteMsg(conn, hello); err != nil {
		return nil, err
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var resp msg.ServerHelloAuto
	if err := msg.ReadMsgInto(conn, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *autoTransportManager) setAutoAuth(hello *msg.ClientHelloAuto) error {
	login := &msg.Login{
		Timestamp: time.Now().Unix(),
	}
	if err := m.auth.Setter.SetLogin(login); err != nil {
		return err
	}
	hello.Timestamp = login.Timestamp
	hello.PrivilegeKey = login.PrivilegeKey
	return nil
}

func (m *autoTransportManager) setProbeAuth(probe *msg.ProbeTransport) error {
	login := &msg.Login{
		Timestamp: time.Now().Unix(),
	}
	if err := m.auth.Setter.SetLogin(login); err != nil {
		return err
	}
	probe.Timestamp = login.Timestamp
	probe.PrivilegeKey = login.PrivilegeKey
	return nil
}

func (m *autoTransportManager) clientCandidates() []string {
	candidates := append([]string(nil), m.common.Transport.Auto.Candidates...)
	if m.common.Transport.ProxyURL != "" {
		return []string{v1.TransportProtocolTCP}
	}
	return candidates
}

func (m *autoTransportManager) activeBlacklistProtocols() []string {
	now := time.Now()
	out := make([]string, 0, len(m.blacklistUntil))
	for protocol, until := range m.blacklistUntil {
		if now.Before(until) {
			out = append(out, protocol)
		}
	}
	sort.Strings(out)
	return out
}

func (m *autoTransportManager) buildCandidates(serverHello *msg.ServerHelloAuto) []autoTransportCandidate {
	clientCandidates := m.clientCandidates()
	clientIndex := make(map[string]int, len(clientCandidates))
	for i, protocol := range clientCandidates {
		if _, ok := clientIndex[protocol]; !ok {
			clientIndex[protocol] = i
		}
	}
	preferIndex := make(map[string]int, len(serverHello.PreferOrder))
	for i, protocol := range serverHello.PreferOrder {
		preferIndex[protocol] = i
	}

	candidates := make([]autoTransportCandidate, 0, len(serverHello.Transports))
	for _, ep := range serverHello.Transports {
		if !ep.Enabled {
			continue
		}
		localRank, ok := clientIndex[ep.Protocol]
		if !ok {
			continue
		}
		if !m.allowUDP() && isUDPTransportProtocol(ep.Protocol) {
			continue
		}
		if m.isBlacklisted(ep.Protocol) {
			continue
		}
		serverRank, ok := preferIndex[ep.Protocol]
		if !ok {
			serverRank = len(serverHello.PreferOrder)
		}
		candidates = append(candidates, autoTransportCandidate{
			Protocol: ep.Protocol,
			Addr:     m.usableEndpointAddr(ep.Addr),
			Port:     ep.Port,
			Priority: serverRank*100 + localRank,
			Source:   "server",
		})
	}

	if len(candidates) == 0 && len(m.blacklistUntil) > 0 {
		for protocol := range m.blacklistUntil {
			delete(m.blacklistUntil, protocol)
		}
		return m.buildCandidates(serverHello)
	}

	slices.SortFunc(candidates, func(a, b autoTransportCandidate) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		return slices.Index(clientCandidates, a.Protocol) - slices.Index(clientCandidates, b.Protocol)
	})
	return candidates
}

func (m *autoTransportManager) fixedModeCandidatesLocked(
	candidates []autoTransportCandidate,
) []autoTransportCandidate {
	if m.selected != nil {
		for _, candidate := range candidates {
			if sameAutoTransportCandidate(*m.selected, candidate) {
				return []autoTransportCandidate{candidate}
			}
		}
	}
	if m.lastGood != "" {
		for _, candidate := range candidates {
			if candidate.Protocol == m.lastGood {
				return []autoTransportCandidate{candidate}
			}
		}
	}
	return candidates
}

func (m *autoTransportManager) isBlacklisted(protocol string) bool {
	until, ok := m.blacklistUntil[protocol]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(m.blacklistUntil, protocol)
		return false
	}
	return true
}

func (m *autoTransportManager) usableEndpointAddr(addr string) string {
	if addr == "" || addr == "0.0.0.0" || addr == "::" || addr == "[::]" {
		return m.common.ServerAddr
	}
	return addr
}

func (m *autoTransportManager) probeAndSelect(
	ctx context.Context,
	candidates []autoTransportCandidate,
	reason string,
) (autoTransportProbeResult, []autoTransportProbeResult, error) {
	resultsCh := make(chan autoTransportProbeResult, len(candidates))
	for _, candidate := range candidates {
		c := candidate
		go func() {
			resultsCh <- m.probeCandidate(ctx, c)
		}()
	}

	results := make([]autoTransportProbeResult, 0, len(candidates))
	for range candidates {
		results = append(results, <-resultsCh)
	}

	successes := make([]autoTransportProbeResult, 0, len(results))
	for _, result := range results {
		if result.Successes > 0 {
			successes = append(successes, result)
		}
	}
	if len(successes) == 0 {
		return autoTransportProbeResult{}, results, fmt.Errorf("auto transport: all candidates failed probing")
	}

	sort.Slice(successes, func(i, j int) bool {
		if successes[i].Score != successes[j].Score {
			return successes[i].Score > successes[j].Score
		}
		return successes[i].Candidate.Priority < successes[j].Candidate.Priority
	})
	return m.chooseCandidateByPolicy(reason, successes), results, nil
}

func (m *autoTransportManager) chooseCandidateByPolicy(
	reason string,
	successes []autoTransportProbeResult,
) autoTransportProbeResult {
	best := successes[0]
	if m.selected == nil || isForcedAutoTransportReason(reason) {
		return best
	}

	currentIndex := slices.IndexFunc(successes, func(result autoTransportProbeResult) bool {
		return sameAutoTransportCandidate(*m.selected, result.Candidate)
	})
	if currentIndex < 0 {
		return best
	}

	current := successes[currentIndex]
	if sameAutoTransportCandidate(current.Candidate, best.Candidate) {
		return best
	}
	if m.stickyRemainingSecLocked() > 0 {
		log.Infof("auto transport: keep current %s@%s:%d because sticky is active",
			current.Candidate.Protocol, current.Candidate.Addr, current.Candidate.Port)
		return current
	}
	if m.cooldownRemainingSecLocked() > 0 {
		log.Infof("auto transport: keep current %s@%s:%d because switch cooldown is active",
			current.Candidate.Protocol, current.Candidate.Addr, current.Candidate.Port)
		return current
	}
	if !scoreBeatsHysteresis(best.Score, current.Score) {
		log.Infof("auto transport: keep current %s@%s:%d because best score [%d] does not beat current score [%d] by hysteresis",
			current.Candidate.Protocol, current.Candidate.Addr, current.Candidate.Port, best.Score, current.Score)
		return current
	}
	return best
}

func (m *autoTransportManager) probeCandidate(
	ctx context.Context,
	candidate autoTransportCandidate,
) autoTransportProbeResult {
	result := autoTransportProbeResult{
		Candidate: candidate,
	}
	var totalRTT time.Duration
	for i := 0; i < m.common.Transport.Auto.ProbeCount; i++ {
		rtt, err := m.probeOnce(ctx, candidate)
		if err != nil {
			result.Err = err
			continue
		}
		result.Successes++
		totalRTT += rtt
	}
	if result.Successes > 0 {
		result.AvgRTT = totalRTT / time.Duration(result.Successes)
		strategy := autoTransportStrategyByName(m.common.Transport.Auto.Strategy)
		result.ScoreDetail = strategy.Score(autoTransportScoringInput{
			Candidate:  candidate,
			Successes:  result.Successes,
			ProbeCount: m.common.Transport.Auto.ProbeCount,
			AvgRTT:     result.AvgRTT,
			LastGood:   m.lastGood,
			Failures:   m.failures[candidate.Protocol],
		})
		result.Score = result.ScoreDetail.Total
	}
	return result
}

func (m *autoTransportManager) probeOnce(ctx context.Context, candidate autoTransportCandidate) (time.Duration, error) {
	timeout := time.Duration(m.common.Transport.Auto.ProbeTimeoutMs) * time.Millisecond
	doCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := m.configForCandidate(candidate)
	connector := m.connectorCreator(doCtx, cfg)
	start := time.Now()
	if err := connector.Open(); err != nil {
		return 0, err
	}
	defer connector.Close()

	conn, err := connector.Connect()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	probe := &msg.ProbeTransport{
		Protocol:          candidate.Protocol,
		Addr:              candidate.Addr,
		Port:              candidate.Port,
		ClientAutoVersion: msg.AutoTransportVersion,
	}
	if err := m.setProbeAuth(probe); err != nil {
		return 0, err
	}
	if err := msg.WriteMsg(conn, probe); err != nil {
		return 0, err
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var resp msg.ProbeTransportResp
	if err := msg.ReadMsgInto(conn, &resp); err != nil {
		return 0, err
	}
	if resp.Error != "" {
		return 0, errors.New(resp.Error)
	}
	if resp.ServerAutoVersion != 0 && resp.ServerAutoVersion != msg.AutoTransportVersion {
		return 0, fmt.Errorf("unsupported server auto transport version %d", resp.ServerAutoVersion)
	}
	if resp.Protocol != candidate.Protocol || resp.Port != candidate.Port {
		return 0, fmt.Errorf("probe response mismatch, got %s@:%d", resp.Protocol, resp.Port)
	}
	return time.Since(start), nil
}

func (m *autoTransportManager) recordSelectionLocked(
	candidate autoTransportCandidate,
	score int64,
	dynamic bool,
	reason string,
) {
	if m.selected != nil && !sameAutoTransportCandidate(*m.selected, candidate) {
		m.previousProtocol = m.selected.Protocol
		switchEvent := map[string]any{
			"event":          "transport_switch",
			"old_protocol":   m.selected.Protocol,
			"new_protocol":   candidate.Protocol,
			"reason":         reason,
			"scores":         copyInt64Map(m.lastScores),
			"sticky_expired": m.stickyRemainingSecLocked() == 0,
			"blacklist":      m.activeBlacklistProtocols(),
		}
		if data, err := json.Marshal(switchEvent); err == nil {
			log.Infof("auto transport event: %s", string(data))
		}
		log.Infof("auto transport: transport_switch old_protocol [%s] old_addr [%s] old_port [%d] new_protocol [%s] new_addr [%s] new_port [%d] reason [%s]",
			m.selected.Protocol, m.selected.Addr, m.selected.Port,
			candidate.Protocol, candidate.Addr, candidate.Port, reason)
		m.switchCount++
		m.lastSwitchAt = time.Now()
		m.selectedAt = time.Time{}
	}

	c := candidate
	m.selected = &c
	m.selectedScore = score
	m.lastDynamic = dynamic
	m.lastReason = reason
}

func (m *autoTransportManager) stickyRemainingSecLocked() int64 {
	if m.selectedAt.IsZero() || m.common.Transport.Auto.StickyDurationSec <= 0 {
		return 0
	}
	return remainingSeconds(m.selectedAt, time.Duration(m.common.Transport.Auto.StickyDurationSec)*time.Second)
}

func (m *autoTransportManager) cooldownRemainingSecLocked() int64 {
	if m.lastSwitchAt.IsZero() || m.common.Transport.Auto.CooldownSec <= 0 {
		return 0
	}
	return remainingSeconds(m.lastSwitchAt, time.Duration(m.common.Transport.Auto.CooldownSec)*time.Second)
}

func remainingSeconds(start time.Time, duration time.Duration) int64 {
	remaining := time.Until(start.Add(duration))
	if remaining <= 0 {
		return 0
	}
	return int64((remaining + time.Second - 1) / time.Second)
}

func sameAutoTransportCandidate(a, b autoTransportCandidate) bool {
	return a.Protocol == b.Protocol && a.Addr == b.Addr && a.Port == b.Port
}

func isForcedAutoTransportReason(reason string) bool {
	switch reason {
	case autoTransportReasonControlClosed, autoTransportReasonLoginFailure:
		return true
	default:
		return false
	}
}

func scoreBeatsHysteresis(best, current int64) bool {
	if best <= current {
		return false
	}
	if current <= 0 {
		return best-current >= 1000
	}
	return float64(best) >= float64(current)*autoTransportHysteresisRatio
}

func scoresFromProbeResults(results []autoTransportProbeResult) map[string]int64 {
	out := make(map[string]int64, len(results))
	for _, result := range results {
		if result.Successes == 0 {
			continue
		}
		out[autoTransportCandidateKey(result.Candidate)] = result.Score
	}
	return out
}

func scoreDetailsFromProbeResults(results []autoTransportProbeResult) map[string]AutoTransportScoreDetail {
	out := make(map[string]AutoTransportScoreDetail, len(results))
	for _, result := range results {
		if result.Successes == 0 {
			continue
		}
		out[autoTransportCandidateKey(result.Candidate)] = result.ScoreDetail
	}
	return out
}

func probeStatusFromResults(
	results []autoTransportProbeResult,
	probeCount int,
) (map[string]float64, map[string]int64, map[string]string) {
	if probeCount <= 0 {
		probeCount = 1
	}
	successRates := make(map[string]float64, len(results))
	probeRTTMs := make(map[string]int64, len(results))
	probeErrors := make(map[string]string, len(results))
	for _, result := range results {
		key := autoTransportCandidateKey(result.Candidate)
		successRates[key] = float64(result.Successes) / float64(probeCount)
		if result.Successes > 0 {
			probeRTTMs[key] = result.AvgRTT.Milliseconds()
		}
		if result.Err != nil {
			probeErrors[key] = result.Err.Error()
		}
	}
	return successRates, probeRTTMs, probeErrors
}

func autoTransportCandidateKey(candidate autoTransportCandidate) string {
	return fmt.Sprintf("%s@%s:%d", candidate.Protocol, candidate.Addr, candidate.Port)
}

func (m *autoTransportManager) allowUDP() bool {
	return lo.FromPtr(m.common.Transport.Auto.AllowUDP)
}

func isUDPTransportProtocol(protocol string) bool {
	return protocol == v1.TransportProtocolKCP || protocol == v1.TransportProtocolQUIC
}

func (m *autoTransportManager) persistLastGoodEnabled() bool {
	return lo.FromPtr(m.common.Transport.Auto.PersistLastGood) && m.statePath != ""
}

func (m *autoTransportManager) loadLastGood() {
	if !m.persistLastGoodEnabled() {
		return
	}

	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("auto transport: read persisted state failed: %v", err)
		}
		return
	}

	var state autoTransportPersistState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Warnf("auto transport: parse persisted state failed: %v", err)
		return
	}
	if state.LastGoodProtocol == "" || !slices.Contains(m.clientCandidates(), state.LastGoodProtocol) {
		return
	}
	m.lastGood = state.LastGoodProtocol
	log.Infof("auto transport: loaded persisted last good protocol [%s]", m.lastGood)
}

func (m *autoTransportManager) persistLastGood(protocol string) {
	if !m.persistLastGoodEnabled() || protocol == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(m.statePath), 0o700); err != nil {
		log.Warnf("auto transport: create state directory failed: %v", err)
		return
	}
	data, err := json.MarshalIndent(autoTransportPersistState{LastGoodProtocol: protocol}, "", "  ")
	if err != nil {
		log.Warnf("auto transport: marshal persisted state failed: %v", err)
		return
	}
	if err := os.WriteFile(m.statePath, data, 0o600); err != nil {
		log.Warnf("auto transport: write persisted state failed: %v", err)
	}
}

func (m *autoTransportManager) configForCandidate(candidate autoTransportCandidate) *v1.ClientCommonConfig {
	cfg := cloneClientCommonConfig(m.common)
	cfg.ServerAddr = candidate.Addr
	cfg.ServerPort = candidate.Port
	cfg.Transport.Protocol = candidate.Protocol
	return cfg
}

func cloneClientCommonConfig(in *v1.ClientCommonConfig) *v1.ClientCommonConfig {
	out := *in
	out.Start = append([]string(nil), in.Start...)
	out.IncludeConfigFiles = append([]string(nil), in.IncludeConfigFiles...)
	out.Auth.AdditionalScopes = append([]v1.AuthScope(nil), in.Auth.AdditionalScopes...)
	out.Metadatas = copyStringMap(in.Metadatas)
	out.FeatureGates = copyBoolMap(in.FeatureGates)
	return &out
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyFloat64Map(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyInt64Map(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyScoreDetailsMap(in map[string]AutoTransportScoreDetail) map[string]AutoTransportScoreDetail {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]AutoTransportScoreDetail, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
