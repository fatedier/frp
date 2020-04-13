// Copyright 2018 fatedier, fatedier@gmail.com
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
	"math/rand"
	"time"

	"github.com/fatedier/golib/crypto"

	_ "github.com/fatedier/frp/assets/frps/statik"
	_ "github.com/fatedier/frp/models/metrics"
	"github.com/kardianos/service"
	"os"
	"path/filepath"
)

func main() {
	crypto.DefaultSalt = "frp"
	rand.Seed(time.Now().UnixNano())

		svcConfig := &service.Config{
		Name:	"FRPS",
	}
	prg := &FRPS{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		Execute()
		return
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //chang workdir
	err = os.Chdir(dir)
	_ = s.Run()
}

type FRPS struct {}

func (p *FRPS) Start(s service.Service) error {
	_, _ = s.Status()
	go Execute()
	return nil
}

func (p *FRPS) Stop(s service.Service) error {
	_, _ = s.Status()
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}
