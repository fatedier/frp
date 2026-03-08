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

package group

import (
	"net"
	"strconv"

	"github.com/fatedier/frp/server/ports"
)

// TCPGroupCtl manages all TCPGroups.
type TCPGroupCtl struct {
	groupRegistry[*TCPGroup]
	portManager *ports.Manager
}

// NewTCPGroupCtl returns a new TCPGroupCtl.
func NewTCPGroupCtl(portManager *ports.Manager) *TCPGroupCtl {
	return &TCPGroupCtl{
		groupRegistry: newGroupRegistry[*TCPGroup](),
		portManager:   portManager,
	}
}

// Listen is the wrapper for TCPGroup's Listen.
// If there is no group, one will be created.
func (tgc *TCPGroupCtl) Listen(proxyName string, group string, groupKey string,
	addr string, port int,
) (l net.Listener, realPort int, err error) {
	for {
		tcpGroup := tgc.getOrCreate(group, func() *TCPGroup {
			return NewTCPGroup(tgc)
		})
		l, realPort, err = tcpGroup.Listen(proxyName, group, groupKey, addr, port)
		if err == errGroupStale {
			continue
		}
		return
	}
}

// TCPGroup routes connections to different proxies.
type TCPGroup struct {
	baseGroup

	addr     string
	port     int
	realPort int
	ctl      *TCPGroupCtl
}

// NewTCPGroup returns a new TCPGroup.
func NewTCPGroup(ctl *TCPGroupCtl) *TCPGroup {
	return &TCPGroup{
		ctl: ctl,
	}
}

// Listen will return a new Listener.
// If TCPGroup already has a listener, just add a new Listener to the queues,
// otherwise listen on the real address.
func (tg *TCPGroup) Listen(proxyName string, group string, groupKey string, addr string, port int) (ln *Listener, realPort int, err error) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if !tg.ctl.isCurrent(group, func(cur *TCPGroup) bool { return cur == tg }) {
		return nil, 0, errGroupStale
	}
	if len(tg.lns) == 0 {
		// the first listener, listen on the real address
		realPort, err = tg.ctl.portManager.Acquire(proxyName, port)
		if err != nil {
			return
		}
		tcpLn, errRet := net.Listen("tcp", net.JoinHostPort(addr, strconv.Itoa(realPort)))
		if errRet != nil {
			tg.ctl.portManager.Release(realPort)
			err = errRet
			return
		}

		tg.addr = addr
		tg.port = port
		tg.realPort = realPort
		tg.initBase(group, groupKey, tcpLn, func() {
			tg.ctl.portManager.Release(tg.realPort)
			tg.ctl.removeIf(tg.group, func(cur *TCPGroup) bool {
				return cur == tg
			})
		})
		ln = tg.newListener(tcpLn.Addr())
		go tg.worker(tcpLn, tg.acceptCh)
	} else {
		// address and port in the same group must be equal
		if tg.group != group || tg.addr != addr {
			err = ErrGroupParamsInvalid
			return
		}
		if tg.port != port {
			err = ErrGroupDifferentPort
			return
		}
		if tg.groupKey != groupKey {
			err = ErrGroupAuthFailed
			return
		}
		ln = tg.newListener(tg.lns[0].Addr())
		realPort = tg.realPort
	}
	return
}
