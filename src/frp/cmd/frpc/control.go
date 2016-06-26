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

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"frp/models/client"
	"frp/models/consts"
	"frp/models/msg"
	"frp/utils/conn"
	"frp/utils/log"
	"frp/utils/pcrypto"
)

func ControlProcess(cli *client.ProxyClient, wait *sync.WaitGroup) {
	defer wait.Done()

	msgSendChan := make(chan interface{}, 1024)

	c, err := loginToServer(cli)
	if err != nil {
		log.Error("ProxyName [%s], connect to server failed!", cli.Name)
		return
	}
	defer c.Close()

	go heartbeatSender(c, msgSendChan)

	go msgSender(cli, c, msgSendChan)
	msgReader(cli, c, msgSendChan)

	close(msgSendChan)
}

// loop for reading messages from frpc after control connection is established
func msgReader(cli *client.ProxyClient, c *conn.Conn, msgSendChan chan interface{}) error {
	// for heartbeat
	var heartbeatTimeout bool = false
	timer := time.AfterFunc(time.Duration(client.HeartBeatTimeout)*time.Second, func() {
		heartbeatTimeout = true
		c.Close()
		log.Error("ProxyName [%s], heartbeatRes from frps timeout", cli.Name)
	})
	defer timer.Stop()

	for {
		buf, err := c.ReadLine()
		if err == io.EOF || c == nil || c.IsClosed() {
			c.Close()
			log.Warn("ProxyName [%s], frps close this control conn!", cli.Name)
			var delayTime time.Duration = 1

			// loop until reconnect to frps
			for {
				log.Info("ProxyName [%s], try to reconnect to frps [%s:%d]...", cli.Name, client.ServerAddr, client.ServerPort)
				c, err = loginToServer(cli)
				if err == nil {
					close(msgSendChan)
					msgSendChan = make(chan interface{}, 1024)
					go heartbeatSender(c, msgSendChan)
					go msgSender(cli, c, msgSendChan)
					break
				}

				if delayTime < 60 {
					delayTime = delayTime * 2
				}
				time.Sleep(delayTime * time.Second)
			}
			continue
		} else if err != nil {
			log.Warn("ProxyName [%s], read from frps error: %v", cli.Name, err)
			continue
		}

		ctlRes := &msg.ControlRes{}
		if err := json.Unmarshal([]byte(buf), &ctlRes); err != nil {
			log.Warn("ProxyName [%s], parse msg from frps error: %v : %s", cli.Name, err, buf)
			continue
		}

		switch ctlRes.Type {
		case consts.HeartbeatRes:
			log.Debug("ProxyName [%s], receive heartbeat response", cli.Name)
			timer.Reset(time.Duration(client.HeartBeatTimeout) * time.Second)
		case consts.NoticeUserConn:
			log.Debug("ProxyName [%s], new user connection", cli.Name)
			// join local and remote connections, async
			go cli.StartTunnel(client.ServerAddr, client.ServerPort)
		default:
			log.Warn("ProxyName [%s}, unsupport msgType [%d]", cli.Name, ctlRes.Type)
		}
	}
	return nil
}

// loop for sending messages from channel to frps
func msgSender(cli *client.ProxyClient, c *conn.Conn, msgSendChan chan interface{}) {
	for {
		msg, ok := <-msgSendChan
		if !ok {
			break
		}

		buf, _ := json.Marshal(msg)
		err := c.Write(string(buf) + "\n")
		if err != nil {
			log.Warn("ProxyName [%s], write to server error, proxy exit", cli.Name)
			c.Close()
			break
		}
	}
}

func loginToServer(cli *client.ProxyClient) (c *conn.Conn, err error) {
	c, err = conn.ConnectServer(client.ServerAddr, client.ServerPort)
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", cli.Name, client.ServerAddr, client.ServerPort, err)
		return
	}

	nowTime := time.Now().Unix()
	authKey := pcrypto.GetAuthKey(cli.Name + cli.AuthToken + fmt.Sprintf("%d", nowTime))
	req := &msg.ControlReq{
		Type:          consts.NewCtlConn,
		ProxyName:     cli.Name,
		AuthKey:       authKey,
		UseEncryption: cli.UseEncryption,
		UseGzip:       cli.UseGzip,
		PrivilegeMode: cli.PrivilegeMode,
		ProxyType:     cli.Type,
		Timestamp:     nowTime,
	}
	if cli.PrivilegeMode {
		req.RemotePort = cli.RemotePort
		req.CustomDomains = cli.CustomDomains
	}

	buf, _ := json.Marshal(req)
	err = c.Write(string(buf) + "\n")
	if err != nil {
		log.Error("ProxyName [%s], write to server error, %v", cli.Name, err)
		return
	}

	res, err := c.ReadLine()
	if err != nil {
		log.Error("ProxyName [%s], read from server error, %v", cli.Name, err)
		return
	}
	log.Debug("ProxyName [%s], read [%s]", cli.Name, res)

	ctlRes := &msg.ControlRes{}
	if err = json.Unmarshal([]byte(res), &ctlRes); err != nil {
		log.Error("ProxyName [%s], format server response error, %v", cli.Name, err)
		return
	}

	if ctlRes.Code != 0 {
		log.Error("ProxyName [%s], start proxy error, %s", cli.Name, ctlRes.Msg)
		return c, fmt.Errorf("%s", ctlRes.Msg)
	}

	log.Debug("ProxyName [%s], connect to server [%s:%d] success!", cli.Name, client.ServerAddr, client.ServerPort)
	return
}

func heartbeatSender(c *conn.Conn, msgSendChan chan interface{}) {
	heartbeatReq := &msg.ControlReq{
		Type: consts.HeartbeatReq,
	}
	log.Info("Start to send heartbeat to frps")
	for {
		time.Sleep(time.Duration(client.HeartBeatInterval) * time.Second)
		if c != nil && !c.IsClosed() {
			log.Debug("Send heartbeat to server")
			msgSendChan <- heartbeatReq
		} else {
			break
		}
	}
	log.Debug("Heartbeat goroutine exit")
}
