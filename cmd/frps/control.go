package main

import (
	"encoding/json"
	"fmt"

	"github.com/fatedier/frp/pkg/models"
	"github.com/fatedier/frp/pkg/utils/conn"
	"github.com/fatedier/frp/pkg/utils/log"
)

func ProcessControlConn(l *conn.Listener) {
	for {
		c := l.GetConn()
		log.Debug("Get one new conn, %v", c.GetRemoteAddr())
		go controlWorker(c)
	}
}

// control connection from every client and server
func controlWorker(c *conn.Conn) {
	// the first message is from client to server
	// if error, close connection
	res, err := c.ReadLine()
	if err != nil {
		log.Warn("Read error, %v", err)
		return
	}
	log.Debug("get: %s", res)

	clientCtlReq := &models.ClientCtlReq{}
	clientCtlRes := &models.ClientCtlRes{}
	if err := json.Unmarshal([]byte(res), &clientCtlReq); err != nil {
		log.Warn("Parse err: %v : %s", err, res)
		return
	}

	// check
	succ, msg, needRes := checkProxy(clientCtlReq, c)
	if !succ {
		clientCtlRes.Code = 1
		clientCtlRes.Msg = msg
	}

	if needRes {
		buf, _ := json.Marshal(clientCtlRes)
		err = c.Write(string(buf) + "\n")
		if err != nil {
			log.Warn("Write error, %v", err)
		}
	} else {
		// work conn, just return
		return
	}

	defer c.Close()
	// others is from server to client
	server, ok := ProxyServers[clientCtlReq.ProxyName]
	if !ok {
		log.Warn("ProxyName [%s] is not exist", clientCtlReq.ProxyName)
		return
	}

	serverCtlReq := &models.ClientCtlReq{}
	serverCtlReq.Type = models.WorkConn
	for {
		server.WaitUserConn()
		buf, _ := json.Marshal(serverCtlReq)
		err = c.Write(string(buf) + "\n")
		if err != nil {
			log.Warn("ProxyName [%s], write to client error, proxy exit", server.Name)
			server.Close()
			return
		}

		log.Debug("ProxyName [%s], write to client to add work conn success", server.Name)
	}

	return
}

func checkProxy(req *models.ClientCtlReq, c *conn.Conn) (succ bool, msg string, needRes bool) {
	succ = false
	needRes = true
	// check if proxy name exist
	server, ok := ProxyServers[req.ProxyName]
	if !ok {
		msg = fmt.Sprintf("ProxyName [%s] is not exist", req.ProxyName)
		log.Warn(msg)
		return
	}

	// check password
	if req.Passwd != server.Passwd {
		msg = fmt.Sprintf("ProxyName [%s], password is not correct", req.ProxyName)
		log.Warn(msg)
		return
	}

	// control conn
	if req.Type == models.ControlConn {
		if server.Status != models.Idle {
			msg = fmt.Sprintf("ProxyName [%s], already in use", req.ProxyName)
			log.Warn(msg)
			return
		}

		// start proxy and listen for user conn, no block
		err := server.Start()
		if err != nil {
			msg = fmt.Sprintf("ProxyName [%s], start proxy error: %v", req.ProxyName, err.Error())
			log.Warn(msg)
			return
		}

		log.Info("ProxyName [%s], start proxy success", req.ProxyName)
	} else if req.Type == models.WorkConn {
		// work conn
		needRes = false
		if server.Status != models.Working {
			log.Warn("ProxyName [%s], is not working when it gets one new work conn", req.ProxyName)
			return
		}

		server.CliConnChan <- c
	} else {
		msg = fmt.Sprintf("ProxyName [%s], type [%d] unsupport", req.ProxyName)
		log.Warn(msg)
		return
	}

	succ = true
	return
}
