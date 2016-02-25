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

func ControlProcess(cli *client.ProxyClient, wait *sync.WaitGroup) {
	defer wait.Done()

	c, err := loginToServer(cli)
	if err != nil {
		log.Error("ProxyName [%s], connect to server failed!", cli.Name)
		return
	}
	defer c.Close()

	for {
		// ignore response content now
		_, err := c.ReadLine()
		if err == io.EOF {
			log.Debug("ProxyName [%s], server close this control conn", cli.Name)
			var sleepTime time.Duration = 1

			// loop until connect to server
			for {
				log.Debug("ProxyName [%s], try to reconnect to server[%s:%d]...", cli.Name, client.ServerAddr, client.ServerPort)
				tmpConn, err := loginToServer(cli)
				if err == nil {
					c.Close()
					c = tmpConn
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
	log.Debug("Start to send heartbeat")
	for {
		time.Sleep(time.Duration(client.HeartBeatInterval) * time.Second)
		if !c.IsClosed() {
			err := c.Write("\n")
			if err != nil {
				log.Error("Send hearbeat to server failed! Err:%s", err.Error())
				continue
			}
		} else {
			break
		}
	}
	log.Debug("heartbeat exit")
}
