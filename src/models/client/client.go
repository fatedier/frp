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
	"sync"
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

	udpTunnel *conn.Conn
	once      sync.Once
}

// if proxy type is udp, keep a tcp connection for transferring udp packages
func (pc *ProxyClient) StartUdpTunnelOnce(addr string, port int64) {
	pc.once.Do(func() {
		var err error
		var c *conn.Conn
		udpProcessor := NewUdpProcesser(nil, pc.LocalIp, pc.LocalPort)
		for {
			if pc.udpTunnel == nil || pc.udpTunnel.IsClosed() {
				if HttpProxy == "" {
					c, err = conn.ConnectServer(fmt.Sprintf("%s:%d", addr, port))
				} else {
					c, err = conn.ConnectServerByHttpProxy(HttpProxy, fmt.Sprintf("%s:%d", addr, port))
				}
				if err != nil {
					log.Error("ProxyName [%s], udp tunnel connect to server [%s:%d] error, %v", pc.Name, addr, port, err)
					time.Sleep(10 * time.Second)
					continue
				}
				log.Info("ProxyName [%s], udp tunnel reconnect to server [%s:%d] success", pc.Name, addr, port)

				nowTime := time.Now().Unix()
				req := &msg.ControlReq{
					Type:          consts.NewWorkConnUdp,
					ProxyName:     pc.Name,
					PrivilegeMode: pc.PrivilegeMode,
					Timestamp:     nowTime,
				}
				if pc.PrivilegeMode == true {
					req.PrivilegeKey = pcrypto.GetAuthKey(pc.Name + PrivilegeToken + fmt.Sprintf("%d", nowTime))
				} else {
					req.AuthKey = pcrypto.GetAuthKey(pc.Name + pc.AuthToken + fmt.Sprintf("%d", nowTime))
				}

				buf, _ := json.Marshal(req)
				err = c.WriteString(string(buf) + "\n")
				if err != nil {
					log.Error("ProxyName [%s], udp tunnel write to server error, %v", pc.Name, err)
					c.Close()
					time.Sleep(1 * time.Second)
					continue
				}
				pc.udpTunnel = c
				udpProcessor.UpdateTcpConn(pc.udpTunnel)
				udpProcessor.Run()
			}
			time.Sleep(1 * time.Second)
		}
	})
}

func (pc *ProxyClient) GetLocalConn() (c *conn.Conn, err error) {
	c, err = conn.ConnectServer(fmt.Sprintf("%s:%d", pc.LocalIp, pc.LocalPort))
	if err != nil {
		log.Error("ProxyName [%s], connect to local port error, %v", pc.Name, err)
	}
	return
}

func (pc *ProxyClient) GetRemoteConn(addr string, port int64) (c *conn.Conn, err error) {
	defer func() {
		if err != nil && c != nil {
			c.Close()
		}
	}()

	if HttpProxy == "" {
		c, err = conn.ConnectServer(fmt.Sprintf("%s:%d", addr, port))
	} else {
		c, err = conn.ConnectServerByHttpProxy(HttpProxy, fmt.Sprintf("%s:%d", addr, port))
	}
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", pc.Name, addr, port, err)
		return
	}

	nowTime := time.Now().Unix()
	req := &msg.ControlReq{
		Type:          consts.NewWorkConn,
		ProxyName:     pc.Name,
		PrivilegeMode: pc.PrivilegeMode,
		Timestamp:     nowTime,
	}
	if pc.PrivilegeMode == true {
		req.PrivilegeKey = pcrypto.GetAuthKey(pc.Name + PrivilegeToken + fmt.Sprintf("%d", nowTime))
	} else {
		req.AuthKey = pcrypto.GetAuthKey(pc.Name + pc.AuthToken + fmt.Sprintf("%d", nowTime))
	}

	buf, _ := json.Marshal(req)
	err = c.WriteString(string(buf) + "\n")
	if err != nil {
		log.Error("ProxyName [%s], write to server error, %v", pc.Name, err)
		return
	}

	err = nil
	return
}

func (pc *ProxyClient) StartTunnel(serverAddr string, serverPort int64) (err error) {
	localConn, err := pc.GetLocalConn()
	if err != nil {
		return
	}
	remoteConn, err := pc.GetRemoteConn(serverAddr, serverPort)
	if err != nil {
		return
	}

	// l means local, r means remote
	log.Debug("Join two connections, (l[%s] r[%s]) (l[%s] r[%s])", localConn.GetLocalAddr(), localConn.GetRemoteAddr(),
		remoteConn.GetLocalAddr(), remoteConn.GetRemoteAddr())
	needRecord := false
	go msg.JoinMore(localConn, remoteConn, pc.BaseConf, needRecord)

	return nil
}
