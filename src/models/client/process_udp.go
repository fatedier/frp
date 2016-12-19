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
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/src/models/msg"
	"github.com/fatedier/frp/src/utils/conn"
	"github.com/fatedier/frp/src/utils/pool"
)

type UdpProcesser struct {
	tcpConn *conn.Conn
	closeCh chan struct{}

	localAddr string

	// cache local udp connections
	// key is remoteAddr
	localUdpConns map[string]*net.UDPConn
	mutex         sync.RWMutex
	tcpConnMutex  sync.RWMutex
}

func NewUdpProcesser(c *conn.Conn, localIp string, localPort int64) *UdpProcesser {
	return &UdpProcesser{
		tcpConn:       c,
		closeCh:       make(chan struct{}),
		localAddr:     fmt.Sprintf("%s:%d", localIp, localPort),
		localUdpConns: make(map[string]*net.UDPConn),
	}
}

func (up *UdpProcesser) UpdateTcpConn(c *conn.Conn) {
	up.tcpConnMutex.Lock()
	defer up.tcpConnMutex.Unlock()
	up.tcpConn = c
}

func (up *UdpProcesser) Run() {
	go up.ReadLoop()
}

func (up *UdpProcesser) ReadLoop() {
	var (
		buf string
		err error
	)
	for {
		udpPacket := &msg.UdpPacket{}

		// read udp package from frps
		buf, err = up.tcpConn.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			} else {
				continue
			}
		}
		err = udpPacket.UnPack([]byte(buf))
		if err != nil {
			continue
		}

		// write to local udp port
		sendConn, ok := up.GetUdpConn(udpPacket.SrcStr)
		if !ok {
			dstAddr, err := net.ResolveUDPAddr("udp", up.localAddr)
			if err != nil {
				continue
			}
			sendConn, err = net.DialUDP("udp", nil, dstAddr)
			if err != nil {
				continue
			}

			up.SetUdpConn(udpPacket.SrcStr, sendConn)
		}

		_, err = sendConn.Write(udpPacket.Content)
		if err != nil {
			sendConn.Close()
			continue
		}

		if !ok {
			go up.Forward(udpPacket, sendConn)
		}
	}
}

func (up *UdpProcesser) Forward(udpPacket *msg.UdpPacket, singleConn *net.UDPConn) {
	addr := udpPacket.SrcStr
	defer up.RemoveUdpConn(addr)

	buf := pool.GetBuf(2048)
	for {
		singleConn.SetReadDeadline(time.Now().Add(120 * time.Second))
		n, remoteAddr, err := singleConn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		// forward to frps
		forwardPacket := msg.NewUdpPacket(buf[0:n], remoteAddr, udpPacket.Src)
		up.tcpConnMutex.RLock()
		err = up.tcpConn.WriteString(string(forwardPacket.Pack()) + "\n")
		up.tcpConnMutex.RUnlock()
		if err != nil {
			return
		}
	}
}

func (up *UdpProcesser) GetUdpConn(addr string) (singleConn *net.UDPConn, ok bool) {
	up.mutex.RLock()
	defer up.mutex.RUnlock()
	singleConn, ok = up.localUdpConns[addr]
	return
}

func (up *UdpProcesser) SetUdpConn(addr string, conn *net.UDPConn) {
	up.mutex.Lock()
	defer up.mutex.Unlock()
	up.localUdpConns[addr] = conn
}

func (up *UdpProcesser) RemoveUdpConn(addr string) {
	up.mutex.Lock()
	defer up.mutex.Unlock()
	if c, ok := up.localUdpConns[addr]; ok {
		c.Close()
	}
	delete(up.localUdpConns, addr)
}
