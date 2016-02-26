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
)

var connection *conn.Conn = nil
var heartBeatTimer *time.Timer = nil

func ControlProcess(cli *client.ProxyClient, wait *sync.WaitGroup) {
	defer wait.Done()

	c, err := loginToServer(cli)
	if err != nil {
		log.Error("ProxyName [%s], connect to server failed!", cli.Name)
		return
	}
	connection = c
	defer connection.Close()

	for {
		// ignore response content now
		content, err := connection.ReadLine()
		if err == io.EOF || nil == connection || connection.IsClosed() {
			log.Debug("ProxyName [%s], server close this control conn", cli.Name)
			var sleepTime time.Duration = 1

			// loop until connect to server
			for {
				log.Debug("ProxyName [%s], try to reconnect to server[%s:%d]...", cli.Name, client.ServerAddr, client.ServerPort)
				tmpConn, err := loginToServer(cli)
				if err == nil {
					connection.Close()
					connection = tmpConn
					break
				}

				if sleepTime < 60 {
					sleepTime = sleepTime * 2
				}
				time.Sleep(sleepTime * time.Second)
			}
			continue
		} else if err != nil {
			log.Warn("ProxyName [%s], read from server error, %v", cli.Name, err)
			continue
		}

		clientCtlRes := &msg.ClientCtlRes{}
		if err := json.Unmarshal([]byte(content), clientCtlRes); err != nil {
			log.Warn("Parse err: %v : %s", err, content)
			continue
		}
		if consts.SCHeartBeatRes == clientCtlRes.GeneralRes.Code {
			if heartBeatTimer != nil {
				log.Debug("Client rcv heartbeat response")
				heartBeatTimer.Reset(time.Duration(client.HeartBeatTimeout) * time.Second)
			} else {
				log.Error("heartBeatTimer is nil")
			}
			continue
		}

		cli.StartTunnel(client.ServerAddr, client.ServerPort)
	}
}

func loginToServer(cli *client.ProxyClient) (c *conn.Conn, err error) {
	c, err = conn.ConnectServer(client.ServerAddr, client.ServerPort)
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", cli.Name, client.ServerAddr, client.ServerPort, err)
		return
	}

	req := &msg.ClientCtlReq{
		Type:      consts.CtlConn,
		ProxyName: cli.Name,
		Passwd:    cli.Passwd,
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

	clientCtlRes := &msg.ClientCtlRes{}
	if err = json.Unmarshal([]byte(res), &clientCtlRes); err != nil {
		log.Error("ProxyName [%s], format server response error, %v", cli.Name, err)
		return
	}

	if clientCtlRes.Code != 0 {
		log.Error("ProxyName [%s], start proxy error, %s", cli.Name, clientCtlRes.Msg)
		return c, fmt.Errorf("%s", clientCtlRes.Msg)
	}

	go startHeartBeat(c)
	log.Debug("ProxyName [%s], connect to server[%s:%d] success!", cli.Name, client.ServerAddr, client.ServerPort)

	return
}

func startHeartBeat(c *conn.Conn) {
	f := func() {
		log.Error("HeartBeat timeout!")
		if c != nil {
			c.Close()
		}
	}
	heartBeatTimer = time.AfterFunc(time.Duration(client.HeartBeatTimeout)*time.Second, f)
	defer heartBeatTimer.Stop()

	clientCtlReq := &msg.ClientCtlReq{
		Type:      consts.CSHeartBeatReq,
		ProxyName: "",
		Passwd:    "",
	}
	request, err := json.Marshal(clientCtlReq)
	if err != nil {
		log.Warn("Serialize clientCtlReq err! Err: %v", err)
	}

	log.Debug("Start to send heartbeat")
	for {
		time.Sleep(time.Duration(client.HeartBeatInterval) * time.Second)
		if c != nil && !c.IsClosed() {
			err = c.Write(string(request) + "\n")
			if err != nil {
				log.Error("Send hearbeat to server failed! Err:%v", err)
				continue
			}
		} else {
			break
		}
	}
	log.Debug("Heartbeat exit")
}
