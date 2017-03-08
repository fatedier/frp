// Copyright 2017 fatedier, fatedier@gmail.com
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

package shutdown

import (
	"sync"
)

type Shutdown struct {
	doing  bool
	ending bool
	start  chan struct{}
	down   chan struct{}
	mu     sync.Mutex
}

func New() *Shutdown {
	return &Shutdown{
		doing:  false,
		ending: false,
		start:  make(chan struct{}),
		down:   make(chan struct{}),
	}
}

func (s *Shutdown) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.doing {
		s.doing = true
		close(s.start)
	}
}

func (s *Shutdown) WaitStart() {
	<-s.start
}

func (s *Shutdown) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ending {
		s.ending = true
		close(s.down)
	}
}

func (s *Shutdown) WaitDown() {
	<-s.down
}
