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
	"slices"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

type AutoTransportScoreDetail struct {
	Strategy        string  `json:"strategy,omitempty"`
	Total           int64   `json:"total"`
	Successes       int     `json:"successes"`
	ProbeCount      int     `json:"probeCount"`
	SuccessRate     float64 `json:"successRate"`
	AvgRTTMs        int64   `json:"avgRTTMs"`
	Priority        int     `json:"priority"`
	SuccessScore    int64   `json:"successScore"`
	LatencyPenalty  int64   `json:"latencyPenalty"`
	PriorityPenalty int64   `json:"priorityPenalty"`
	LastGoodBonus   int64   `json:"lastGoodBonus,omitempty"`
	FailurePenalty  int64   `json:"failurePenalty,omitempty"`
}

type autoTransportScoringInput struct {
	Candidate  autoTransportCandidate
	Successes  int
	ProbeCount int
	AvgRTT     time.Duration
	LastGood   string
	Failures   int
}

type autoTransportScoreStrategy interface {
	Name() string
	Score(autoTransportScoringInput) AutoTransportScoreDetail
}

type weightedAutoTransportStrategy struct {
	name           string
	successWeight  int64
	latencyWeight  int64
	priorityWeight int64
	lastGoodBonus  int64
	failurePenalty int64
}

func (s weightedAutoTransportStrategy) Name() string {
	return s.name
}

func (s weightedAutoTransportStrategy) Score(input autoTransportScoringInput) AutoTransportScoreDetail {
	probeCount := input.ProbeCount
	if probeCount <= 0 {
		probeCount = 1
	}

	detail := AutoTransportScoreDetail{
		Strategy:        s.name,
		Successes:       input.Successes,
		ProbeCount:      probeCount,
		SuccessRate:     float64(input.Successes) / float64(probeCount),
		AvgRTTMs:        input.AvgRTT.Milliseconds(),
		Priority:        input.Candidate.Priority,
		SuccessScore:    int64(input.Successes) * s.successWeight,
		LatencyPenalty:  input.AvgRTT.Milliseconds() * s.latencyWeight,
		PriorityPenalty: int64(input.Candidate.Priority) * s.priorityWeight,
		FailurePenalty:  int64(input.Failures) * s.failurePenalty,
	}
	if input.Candidate.Protocol == input.LastGood {
		detail.LastGoodBonus = s.lastGoodBonus
	}
	detail.Total = detail.SuccessScore -
		detail.LatencyPenalty -
		detail.PriorityPenalty +
		detail.LastGoodBonus -
		detail.FailurePenalty
	return detail
}

var autoTransportScoreStrategies = map[string]autoTransportScoreStrategy{
	v1.AutoTransportStrategyBalanced: weightedAutoTransportStrategy{
		name:           v1.AutoTransportStrategyBalanced,
		successWeight:  10000,
		latencyWeight:  1,
		priorityWeight: 100,
		lastGoodBonus:  500,
		failurePenalty: 300,
	},
	v1.AutoTransportStrategyLatency: weightedAutoTransportStrategy{
		name:           v1.AutoTransportStrategyLatency,
		successWeight:  10000,
		latencyWeight:  5,
		priorityWeight: 40,
		lastGoodBonus:  150,
		failurePenalty: 300,
	},
	v1.AutoTransportStrategyStability: weightedAutoTransportStrategy{
		name:           v1.AutoTransportStrategyStability,
		successWeight:  12000,
		latencyWeight:  1,
		priorityWeight: 70,
		lastGoodBonus:  1200,
		failurePenalty: 1000,
	},
}

func autoTransportStrategyByName(name string) autoTransportScoreStrategy {
	if strategy, ok := autoTransportScoreStrategies[name]; ok {
		return strategy
	}
	return autoTransportScoreStrategies[v1.AutoTransportStrategyBalanced]
}

func supportedAutoTransportScoreStrategyNames() []string {
	names := make([]string, 0, len(autoTransportScoreStrategies))
	for name := range autoTransportScoreStrategies {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
