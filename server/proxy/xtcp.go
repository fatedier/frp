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

package proxy

import (
	"fmt"

	"github.com/fatedier/golib/errors"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/msg"
)

type XTCPProxy struct {
	*BaseProxy
	cfg *config.XTCPProxyConf

	closeCh chan struct{}
}

func (pxy *XTCPProxy) Run() (remoteAddr string, err error) {
	xl := pxy.xl

	if pxy.rc.NatHoleController == nil {
		xl.Error("udp port for xtcp is not specified.")
		err = fmt.Errorf("xtcp is not supported in frps")
		return
	}
	sidCh := pxy.rc.NatHoleController.ListenClient(pxy.GetName(), pxy.cfg.Sk)
	go func() {
		for {
			select {
			case <-pxy.closeCh:
				break
			case sidRequest := <-sidCh:
				sr := sidRequest
				workConn, errRet := pxy.GetWorkConnFromPool(nil, nil)
				if errRet != nil {
					continue
				}
				m := &msg.NatHoleSid{
					Sid: sr.Sid,
				}
				errRet = msg.WriteMsg(workConn, m)
				if errRet != nil {
					xl.Warn("write nat hole sid package error, %v", errRet)
					workConn.Close()
					break
				}

				go func() {
					raw, errRet := msg.ReadMsg(workConn)
					if errRet != nil {
						xl.Warn("read nat hole client ok package error: %v", errRet)
						workConn.Close()
						return
					}
					if _, ok := raw.(*msg.NatHoleClientDetectOK); !ok {
						xl.Warn("read nat hole client ok package format error")
						workConn.Close()
						return
					}

					select {
					case sr.NotifyCh <- struct{}{}:
					default:
					}
				}()
			}
		}
	}()
	return
}

func (pxy *XTCPProxy) GetConf() config.ProxyConf {
	return pxy.cfg
}

func (pxy *XTCPProxy) Close() {
	pxy.BaseProxy.Close()
	pxy.rc.NatHoleController.CloseClient(pxy.GetName())
	_ = errors.PanicToError(func() {
		close(pxy.closeCh)
	})
}
