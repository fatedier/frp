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

package net

import (
	"net"

	"github.com/fatedier/frp/utils/log"
)

type Listener interface {
	Accept() (Conn, error)
	Close() error
	log.Logger
}

type LogListener struct {
	l net.Listener
	net.Listener
	log.Logger
}

func WrapLogListener(l net.Listener) Listener {
	return &LogListener{
		l:        l,
		Listener: l,
		Logger:   log.NewPrefixLogger(""),
	}
}

func (logL *LogListener) Accept() (Conn, error) {
	c, err := logL.l.Accept()
	return WrapConn(c), err
}
