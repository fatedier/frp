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
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/samber/lo"
)

var (
	// mode 0, both EasyNAT, PublicNetwork is always receiver
	// sender | receiver, ttl 7
	// receiver, ttl 7 | sender
	// sender | receiver, ttl 4
	// receiver, ttl 4 | sender
	// sender | receiver
	// receiver | sender
	// sender, sendDelayMs 5000 | receiver
	// sender, sendDelayMs 10000 | receiver
	// receiver | sender, sendDelayMs 5000
	// receiver | sender, sendDelayMs 10000
	mode0Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 7}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 4}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 5000}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 10000}, RecommandBehavior{Role: DetectRoleReceiver}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 5000}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver}, RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 10000}),
	}

	// mode 1, HardNAT is sender, EasyNAT is receiver, port changes is regular
	// sender | receiver, ttl 7, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, ttl 7, portsRangeNumber max 10
	// sender | receiver, ttl 4, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, ttl 4, portsRangeNumber max 10
	// sender | receiver, portsRangeNumber max 10
	// sender, sendDelayMs 2000 | receiver, portsRangeNumber max 10
	mode1Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, SendDelayMs: 2000}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
	}

	// mode 2, HardNAT is receiver, EasyNAT is sender
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports, ttl 7
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports, ttl 4
	// sender, portsRandomNumber 1000, sendDelayMs 3000 | receiver, listen 256 ports
	mode2Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 7},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 4},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256},
		),
	}

	// mode 3, For HardNAT & HardNAT, both changes in the ports are regular
	// sender, portsRangeNumber 10 | receiver, ttl 7, portsRangeNumber 10
	// sender, portsRangeNumber 10 | receiver, ttl 4, portsRangeNumber 10
	// sender, portsRangeNumber 10 | receiver, portsRangeNumber 10
	// receiver, ttl 7, portsRangeNumber 10 | sender, portsRangeNumber 10
	// receiver, ttl 4, portsRangeNumber 10 | sender, portsRangeNumber 10
	// receiver, portsRangeNumber 10 | sender, portsRangeNumber 10
	mode3Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 7, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, TTL: 4, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
		lo.T2(RecommandBehavior{Role: DetectRoleReceiver, PortsRangeNumber: 10}, RecommandBehavior{Role: DetectRoleSender, PortsRangeNumber: 10}),
	}

	// mode 4, Regular ports changes are usually the sender.
	// sender, portsRandomNumber 1000, sendDelayMs: 2000 | receiver, listen 256 ports, ttl 7, portsRangeNumber 2
	// sender, portsRandomNumber 1000, sendDelayMs: 2000 | receiver, listen 256 ports, ttl 4, portsRangeNumber 2
	// sender, portsRandomNumber 1000, SendDelayMs: 2000 | receiver, listen 256 ports, portsRangeNumber 2
	mode4Behaviors = []lo.Tuple2[RecommandBehavior, RecommandBehavior]{
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 7, PortsRangeNumber: 2},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, TTL: 4, PortsRangeNumber: 2},
		),
		lo.T2(
			RecommandBehavior{Role: DetectRoleSender, PortsRandomNumber: 1000, SendDelayMs: 3000},
			RecommandBehavior{Role: DetectRoleReceiver, ListenRandomPorts: 256, PortsRangeNumber: 2},
		),
	}
)

func getBehaviorByMode(mode int) []lo.Tuple2[RecommandBehavior, RecommandBehavior] {
	switch mode {
	case 0:
		return mode0Behaviors
	case 1:
		return mode1Behaviors
	case 2:
		return mode2Behaviors
	case 3:
		return mode3Behaviors
	case 4:
		return mode4Behaviors
	}
	// default
	return mode0Behaviors
}

func getBehaviorByModeAndIndex(mode int, index int) (RecommandBehavior, RecommandBehavior) {
	behaviors := getBehaviorByMode(mode)
	if index >= len(behaviors) {
		return RecommandBehavior{}, RecommandBehavior{}
	}
	return behaviors[index].A, behaviors[index].B
}

func getBehaviorScoresByMode(mode int, defaultScore int) []*BehaviorScore {
	return getBehaviorScoresByMode2(mode, defaultScore, defaultScore)
}

func getBehaviorScoresByMode2(mode int, senderScore, receiverScore int) []*BehaviorScore {
	behaviors := getBehaviorByMode(mode)
	scores := make([]*BehaviorScore, 0, len(behaviors))
	for i := 0; i < len(behaviors); i++ {
		score := receiverScore
		if behaviors[i].A.Role == DetectRoleSender {
			score = senderScore
		}
		scores = append(scores, &BehaviorScore{Mode: mode, Index: i, Score: score})
	}
	return scores
}

type RecommandBehavior struct {
	Role              string
	TTL               int
	SendDelayMs       int
	PortsRangeNumber  int
	PortsRandomNumber int
	ListenRandomPorts int
}

type MakeHoleRecords struct {
	mu             sync.Mutex
	scores         []*BehaviorScore
	LastUpdateTime time.Time
}

func NewMakeHoleRecords(c, v *NatFeature) *MakeHoleRecords {
	scores := []*BehaviorScore{}
	easyCount, hardCount, portsChangedRegularCount := ClassifyFeatureCount([]*NatFeature{c, v})
	appendMode0 := func() {
		switch {
		case c.PublicNetwork:
			scores = append(scores, getBehaviorScoresByMode2(DetectMode0, 0, 1)...)
		case v.PublicNetwork:
			scores = append(scores, getBehaviorScoresByMode2(DetectMode0, 1, 0)...)
		default:
			scores = append(scores, getBehaviorScoresByMode(DetectMode0, 0)...)
		}
	}

	switch {
	case easyCount == 2:
		appendMode0()
	case hardCount == 1 && portsChangedRegularCount == 1:
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode2, 0)...)
		appendMode0()
	case hardCount == 1 && portsChangedRegularCount == 0:
		scores = append(scores, getBehaviorScoresByMode(DetectMode2, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 0)...)
		appendMode0()
	case hardCount == 2 && portsChangedRegularCount == 2:
		scores = append(scores, getBehaviorScoresByMode(DetectMode3, 0)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode4, 0)...)
	case hardCount == 2 && portsChangedRegularCount == 1:
		scores = append(scores, getBehaviorScoresByMode(DetectMode4, 0)...)
	default:
		// hard to make hole, just trying it out.
		scores = append(scores, getBehaviorScoresByMode(DetectMode0, 1)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode1, 1)...)
		scores = append(scores, getBehaviorScoresByMode(DetectMode3, 1)...)
	}
	return &MakeHoleRecords{scores: scores, LastUpdateTime: time.Now()}
}

func (mhr *MakeHoleRecords) ReportSuccess(mode int, index int) {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()
	mhr.LastUpdateTime = time.Now()
	for i := range mhr.scores {
		score := mhr.scores[i]
		if score.Mode != mode || score.Index != index {
			continue
		}

		score.Score += 2
		score.Score = min(score.Score, 10)
		return
	}
}

func (mhr *MakeHoleRecords) Recommand() (mode, index int) {
	mhr.mu.Lock()
	defer mhr.mu.Unlock()

	if len(mhr.scores) == 0 {
		return 0, 0
	}
	maxScore := slices.MaxFunc(mhr.scores, func(a, b *BehaviorScore) int {
		return cmp.Compare(a.Score, b.Score)
	})
	maxScore.Score--
	mhr.LastUpdateTime = time.Now()
	return maxScore.Mode, maxScore.Index
}

type BehaviorScore struct {
	Mode  int
	Index int
	// between -10 and 10
	Score int
}

type Analyzer struct {
	// key is client ip + visitor ip
	records             map[string]*MakeHoleRecords
	dataReserveDuration time.Duration

	mu sync.Mutex
}

func NewAnalyzer(dataReserveDuration time.Duration) *Analyzer {
	return &Analyzer{
		records:             make(map[string]*MakeHoleRecords),
		dataReserveDuration: dataReserveDuration,
	}
}

func (a *Analyzer) GetRecommandBehaviors(key string, c, v *NatFeature) (mode, index int, _ RecommandBehavior, _ RecommandBehavior) {
	a.mu.Lock()
	records, ok := a.records[key]
	if !ok {
		records = NewMakeHoleRecords(c, v)
		a.records[key] = records
	}
	a.mu.Unlock()

	mode, index = records.Recommand()
	cBehavior, vBehavior := getBehaviorByModeAndIndex(mode, index)

	switch mode {
	case DetectMode1:
		// HardNAT is always the sender
		if c.NatType == EasyNAT {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	case DetectMode2:
		// HardNAT is always the receiver
		if c.NatType == HardNAT {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	case DetectMode4:
		// Regular ports changes is always the sender
		if !c.RegularPortsChange {
			cBehavior, vBehavior = vBehavior, cBehavior
		}
	}
	return mode, index, cBehavior, vBehavior
}

func (a *Analyzer) ReportSuccess(key string, mode, index int) {
	a.mu.Lock()
	records, ok := a.records[key]
	a.mu.Unlock()
	if !ok {
		return
	}
	records.ReportSuccess(mode, index)
}

func (a *Analyzer) Clean() (int, int) {
	now := time.Now()
	total := 0
	count := 0

	// cleanup 10w records may take 5ms
	a.mu.Lock()
	defer a.mu.Unlock()
	total = len(a.records)
	// clean up records that have not been used for a period of time.
	for key, records := range a.records {
		if now.Sub(records.LastUpdateTime) > a.dataReserveDuration {
			delete(a.records, key)
			count++
		}
	}
	return count, total
}
