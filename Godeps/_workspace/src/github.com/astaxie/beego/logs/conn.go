// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"encoding/json"
	"io"
	"log"
	"net"
)

// ConnWriter implements LoggerInterface.
// it writes messages in keep-live tcp connection.
type ConnWriter struct {
	lg             *log.Logger
	innerWriter    io.WriteCloser
	ReconnectOnMsg bool   `json:"reconnectOnMsg"`
	Reconnect      bool   `json:"reconnect"`
	Net            string `json:"net"`
	Addr           string `json:"addr"`
	Level          int    `json:"level"`
}

// create new ConnWrite returning as LoggerInterface.
func NewConn() LoggerInterface {
	conn := new(ConnWriter)
	conn.Level = LevelTrace
	return conn
}

// init connection writer with json config.
// json config only need key "level".
func (c *ConnWriter) Init(jsonconfig string) error {
	return json.Unmarshal([]byte(jsonconfig), c)
}

// write message in connection.
// if connection is down, try to re-connect.
func (c *ConnWriter) WriteMsg(msg string, level int) error {
	if level > c.Level {
		return nil
	}
	if c.neddedConnectOnMsg() {
		err := c.connect()
		if err != nil {
			return err
		}
	}

	if c.ReconnectOnMsg {
		defer c.innerWriter.Close()
	}
	c.lg.Println(msg)
	return nil
}

// implementing method. empty.
func (c *ConnWriter) Flush() {

}

// destroy connection writer and close tcp listener.
func (c *ConnWriter) Destroy() {
	if c.innerWriter != nil {
		c.innerWriter.Close()
	}
}

func (c *ConnWriter) connect() error {
	if c.innerWriter != nil {
		c.innerWriter.Close()
		c.innerWriter = nil
	}

	conn, err := net.Dial(c.Net, c.Addr)
	if err != nil {
		return err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
	}

	c.innerWriter = conn
	c.lg = log.New(conn, "", log.Ldate|log.Ltime)
	return nil
}

func (c *ConnWriter) neddedConnectOnMsg() bool {
	if c.Reconnect {
		c.Reconnect = false
		return true
	}

	if c.innerWriter == nil {
		return true
	}

	return c.ReconnectOnMsg
}

func init() {
	Register("conn", NewConn)
}
