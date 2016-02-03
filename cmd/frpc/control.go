package main

import (
	"io"
	"sync"
	"encoding/json"

	"github.com/fatedier/frp/pkg/models"
	"github.com/fatedier/frp/pkg/utils/conn"
	"github.com/fatedier/frp/pkg/utils/log"
)

func ControlProcess(cli *models.ProxyClient, wait *sync.WaitGroup) {
	defer wait.Done()

	c := &conn.Conn{}
	err := c.ConnectServer(ServerAddr, ServerPort)
	if err != nil {
		log.Error("ProxyName [%s], connect to server [%s:%d] error, %v", cli.Name, ServerAddr, ServerPort, err)
		return
	}
	defer c.Close()

	req := &models.ClientCtlReq{
		Type:		models.ControlConn,
		ProxyName:	cli.Name,
		Passwd:		cli.Passwd,
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

	clientCtlRes := &models.ClientCtlRes{}
	if err = json.Unmarshal([]byte(res), &clientCtlRes); err != nil {
		log.Error("ProxyName [%s], format server response error, %v", cli.Name, err)
		return
	}

	if clientCtlRes.Code != 0 {
		log.Error("ProxyName [%s], start proxy error, %s", cli.Name, clientCtlRes.Msg)
		return
	}

	for {
		// ignore response content now
		_, err := c.ReadLine()
		if err == io.EOF {
			log.Debug("ProxyName [%s], server close this control conn", cli.Name)
			break
		} else if err != nil {
			log.Warn("ProxyName [%s], read from server error, %v", cli.Name, err)
			continue
		}

		cli.StartTunnel(ServerAddr, ServerPort)
	}
}
