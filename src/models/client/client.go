// Copyright 2016 fatedier, fatedier@gmail.com
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
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatedier/frp/src/models/config"
	"github.com/fatedier/frp/src/models/consts"
	"github.com/fatedier/frp/src/models/msg"
	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/log"
	"github.com/fatedier/frp/src/utils/pcrypto"
)

type ProxyClient struct {
	config.BaseConf
	LocalIp   string
	LocalPort int64

	RemotePort    int64
	CustomDomains []string
}

func (p *ProxyClient) GetLocalConn() (c *conn.Conn, err error) {
	c, err = conn.ConnectServer(fmt.Sprintf("%s:%d", p.LocalIp, p.LocalPort))
	if err != nil {
		log.Error("ProxyName [%s], connect to local port error, %v", p.Name, err)
	}
	return
}

func (p *ProxyClient) GetRemoteConn(addr string, port int64) (c *conn.Conn, err error) {
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	if HttpProxy == "" {
		c, err = conn.ConnectServer(fmt.Sprintf("%s:%d", addr, port))
	} else {
		c, err = conn.ConnectServerByHttpProxy(HttpProxy, fmt.Sprintf("%s:%d", addr, port))
	}
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", p.Name, addr, port, err)
		return
	}

	nowTime := time.Now().Unix()
	req := &msg.ControlReq{
		Type:          consts.NewWorkConn,
		ProxyName:     p.Name,
		PrivilegeMode: p.PrivilegeMode,
		Timestamp:     nowTime,
	}
	if p.PrivilegeMode == true {
		privilegeKey := pcrypto.GetAuthKey(p.Name + PrivilegeToken + fmt.Sprintf("%d", nowTime))
		req.PrivilegeKey = privilegeKey
	} else {
		authKey := pcrypto.GetAuthKey(p.Name + p.AuthToken + fmt.Sprintf("%d", nowTime))
		req.AuthKey = authKey
	}

	buf, _ := json.Marshal(req)
	err = c.Write(string(buf) + "\n")
	if err != nil {
		log.Error("ProxyName [%s], write to server error, %v", p.Name, err)
		return
	}

	err = nil
	return
}

func (p *ProxyClient) StartTunnel(serverAddr string, serverPort int64) (err error) {
	localConn, err := p.GetLocalConn()
	if err != nil {
		return
	}
	remoteConn, err := p.GetRemoteConn(serverAddr, serverPort)
	if err != nil {
		return
	}

	// l means local, r means remote
	log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", localConn.GetLocalAddr(), localConn.GetRemoteAddr(),
		remoteConn.GetLocalAddr(), remoteConn.GetRemoteAddr())
	needRecord := false
	go msg.JoinMore(localConn, remoteConn, p.BaseConf, needRecord)

	return nil
}
