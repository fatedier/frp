// Copyright 2019 fatedier, fatedier@gmail.com
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

package visitor

import (
	"fmt"
	"io"
	"net"
	"sync"

	frpIo "github.com/fatedier/golib/io"

	frpNet "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/util"
)

// Manager for visitor listeners.
type Manager struct {
	visitorListeners map[string]*frpNet.CustomListener
	skMap            map[string]string

	mu sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		visitorListeners: make(map[string]*frpNet.CustomListener),
		skMap:            make(map[string]string),
	}
}

func (vm *Manager) Listen(name string, sk string) (l *frpNet.CustomListener, err error) {
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

func (vm *Manager) NewConn(name string, conn net.Conn, timestamp int64, signKey string,
	useEncryption bool, useCompression bool,
) (err error) {
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

func (vm *Manager) CloseListener(name string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	delete(vm.visitorListeners, name)
	delete(vm.skMap, name)
}
