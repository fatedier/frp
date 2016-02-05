package main

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"frp/pkg/models"
	"frp/pkg/utils/conn"
	"frp/pkg/utils/log"
)

const (
	heartbeatDuration = 2 //心跳检测时间间隔，单位秒
)

var isHeartBeatContinue bool = true

func ControlProcess(cli *models.ProxyClient, wait *sync.WaitGroup) {
	defer wait.Done()

	c := loginToServer(cli)
	if c == nil {
		log.Error("ProxyName [%s], connect to server failed!", cli.Name)
		return
	}
	defer c.Close()

	for {
		// ignore response content now
		_, err := c.ReadLine()
		if err == io.EOF {
			isHeartBeatContinue = false
			log.Debug("ProxyName [%s], server close this control conn", cli.Name)
			var sleepTime time.Duration = 1
			for {
				log.Debug("ProxyName [%s], try to reconnect to server[%s:%d]...", cli.Name, ServerAddr, ServerPort)
				tmpConn := loginToServer(cli)
				if tmpConn != nil {
					c.Close()
					c = tmpConn
					break
				}

				if sleepTime < 60 {
					sleepTime++
				}
				time.Sleep(sleepTime * time.Second)
			}
			continue
		} else if err != nil {
			log.Warn("ProxyName [%s], read from server error, %v", cli.Name, err)
			continue
		}

		cli.StartTunnel(ServerAddr, ServerPort)
	}
}

func loginToServer(cli *models.ProxyClient) (connection *conn.Conn) {
	c := &conn.Conn{}

	connection = nil
	for i := 0; i < 1; i++ { // ZWF: 此处的for作为控制流使用
		err := c.ConnectServer(ServerAddr, ServerPort)
		if err != nil {
			log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", cli.Name, ServerAddr, ServerPort, err)
			break
		}

		req := &models.ClientCtlReq{
			Type:      models.ControlConn,
			ProxyName: cli.Name,
			Passwd:    cli.Passwd,
		}
		buf, _ := json.Marshal(req)
		err = c.Write(string(buf) + "\n")
		if err != nil {
			log.Error("ProxyName [%s], write to server error, %v", cli.Name, err)
			break
		}

		res, err := c.ReadLine()
		if err != nil {
			log.Error("ProxyName [%s], read from server error, %v", cli.Name, err)
			break
		}
		log.Debug("ProxyName [%s], read [%s]", cli.Name, res)

		clientCtlRes := &models.ClientCtlRes{}
		if err = json.Unmarshal([]byte(res), &clientCtlRes); err != nil {
			log.Error("ProxyName [%s], format server response error, %v", cli.Name, err)
			break
		}

		if clientCtlRes.Code != 0 {
			log.Error("ProxyName [%s], start proxy error, %s", cli.Name, clientCtlRes.Msg)
			break
		}

		connection = c
		go startHeartBeat(connection)
		log.Debug("ProxyName [%s], connect to server[%s:%d] success!", cli.Name, ServerAddr, ServerPort)
	}

	if connection == nil {
		c.Close()
	}

	return
}

func startHeartBeat(con *conn.Conn) {
	isHeartBeatContinue = true
	for {
		time.Sleep(heartbeatDuration * time.Second)
		if isHeartBeatContinue { // 把isHeartBeatContinue放在这里是为了防止SIGPIPE
			err := con.Write("\r\n")
			//log.Debug("send heart beat to server!")
			if err != nil {
				log.Error("Send hearbeat to server failed! Err:%s", err.Error())
			}
		} else {
			break
		}
	}
}
