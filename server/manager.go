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

package server

import (
	"fmt"
	"io"
	"sync"

	frpIo "github.com/fatedier/frp/utils/io"
	frpNet "github.com/fatedier/frp/utils/net"
	"github.com/fatedier/frp/utils/util"
)

type ControlManager struct {
	// controls indexed by run id
	ctlsByRunId map[string]*Control

	mu sync.RWMutex
}

func NewControlManager() *ControlManager {
	return &ControlManager{
		ctlsByRunId: make(map[string]*Control),
	}
}

func (cm *ControlManager) Add(runId string, ctl *Control) (oldCtl *Control) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldCtl, ok := cm.ctlsByRunId[runId]
	if ok {
		oldCtl.Replaced(ctl)
	}
	cm.ctlsByRunId[runId] = ctl
	return
}

func (cm *ControlManager) GetById(runId string) (ctl *Control, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ctl, ok = cm.ctlsByRunId[runId]
	return
}

type ProxyManager struct {
	// proxies indexed by proxy name
	pxys map[string]Proxy

	mu sync.RWMutex
}

func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		pxys: make(map[string]Proxy),
	}
}

func (pm *ProxyManager) Add(name string, pxy Proxy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.pxys[name]; ok {
		return fmt.Errorf("proxy name [%s] is already in use", name)
	}

	pm.pxys[name] = pxy
	return nil
}

func (pm *ProxyManager) Del(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.pxys, name)
}

func (pm *ProxyManager) GetByName(name string) (pxy Proxy, ok bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	pxy, ok = pm.pxys[name]
	return
}

// Manager for visitor listeners.
type VisitorManager struct {
	visitorListeners map[string]*frpNet.CustomListener
	skMap            map[string]string

	mu sync.RWMutex
}

func NewVisitorManager() *VisitorManager {
	return &VisitorManager{
		visitorListeners: make(map[string]*frpNet.CustomListener),
		skMap:            make(map[string]string),
	}
}

func (vm *VisitorManager) Listen(name string, sk string) (l *frpNet.CustomListener, err error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, ok := vm.visitorListeners[name]; ok {
		err = fmt.Errorf("custom listener for [%s] is repeated", name)
		return
	}

	l = frpNet.NewCustomListener()
	vm.visitorListeners[name] = l
	vm.skMap[name] = sk
	return
}

func (vm *VisitorManager) NewConn(name string, conn frpNet.Conn, timestamp int64, signKey string,
	useEncryption bool, useCompression bool) (err error) {

	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if l, ok := vm.visitorListeners[name]; ok {
		var sk string
		if sk = vm.skMap[name]; util.GetAuthKey(sk, timestamp) != signKey {
			err = fmt.Errorf("visitor connection of [%s] auth failed", name)
			return
		}

		var rwc io.ReadWriteCloser = conn
		if useEncryption {
			if rwc, err = frpIo.WithEncryption(rwc, []byte(sk)); err != nil {
				err = fmt.Errorf("create encryption connection failed: %v", err)
				return
			}
		}
		if useCompression {
			rwc = frpIo.WithCompression(rwc)
		}
		err = l.PutConn(frpNet.WrapReadWriteCloserToConn(rwc, conn))
	} else {
		err = fmt.Errorf("custom listener for [%s] doesn't exist", name)
		return
	}
	return
}

func (vm *VisitorManager) CloseListener(name string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	delete(vm.visitorListeners, name)
	delete(vm.skMap, name)
}
