// Copyright 2023 The frp Authors
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

package nathole

import (
	"fmt"
	"net"
	"time"

	"github.com/pion/stun"

	"github.com/fatedier/frp/pkg/msg"
)

var responseTimeout = 3 * time.Second

type Address struct {
	IP   string
	Port int
}

type Message struct {
	Body []byte
	Addr string
}

func Discover(serverAddress string, stunServers []string, key []byte) ([]string, error) {
	// parse address to net.Address
	stunAddresses := make([]net.Addr, 0, len(stunServers))
	for _, stunServer := range stunServers {
		addr, err := net.ResolveUDPAddr("udp4", stunServer)
		if err != nil {
			return nil, err
		}
		stunAddresses = append(stunAddresses, addr)
	}
	serverAddr, err := net.ResolveUDPAddr("udp4", serverAddress)
	if err != nil {
		return nil, err
	}

	// create a discoverConn and get response from messageChan
	discoverConn, err := listen()
	if err != nil {
		return nil, err
	}
	defer discoverConn.Close()

	go discoverConn.readLoop()

	addresses := make([]string, 0, len(stunServers)+1)
	// get external address from frp server
	externalAddr, err := discoverFromServer(discoverConn, serverAddr, key)
	if err != nil {
		return nil, err
	}
	addresses = append(addresses, externalAddr)

	for _, stunAddr := range stunAddresses {
		// get external address from stun server
		externalAddr, err = discoverFromStunServer(discoverConn, stunAddr)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, externalAddr)
	}
	return addresses, nil
}

func discoverFromServer(c *discoverConn, addr net.Addr, key []byte) (string, error) {
	m := &msg.NatHoleBinding{
		TransactionID: NewTransactionID(),
	}

	buf, err := EncodeMessage(m, key)
	if err != nil {
		return "", err
	}

	if _, err := c.conn.WriteTo(buf, addr); err != nil {
		return "", err
	}

	var respMsg msg.NatHoleBindingResp
	select {
	case rawMsg := <-c.messageChan:
		if err := DecodeMessageInto(rawMsg.Body, key, &respMsg); err != nil {
			return "", err
		}
	case <-time.After(responseTimeout):
		return "", fmt.Errorf("wait response from frp server timeout")
	}

	if respMsg.TransactionID == "" {
		return "", fmt.Errorf("error format: no transaction id found")
	}
	if respMsg.Error != "" {
		return "", fmt.Errorf("get externalAddr from frp server error: %s", respMsg.Error)
	}
	return respMsg.Address, nil
}

func discoverFromStunServer(c *discoverConn, addr net.Addr) (string, error) {
	request, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return "", err
	}

	if err = request.NewTransactionID(); err != nil {
		return "", err
	}
	if _, err := c.conn.WriteTo(request.Raw, addr); err != nil {
		return "", err
	}

	var m stun.Message
	select {
	case msg := <-c.messageChan:
		m.Raw = msg.Body
		if err := m.Decode(); err != nil {
			return "", err
		}
	case <-time.After(responseTimeout):
		return "", fmt.Errorf("wait response from stun server timeout")
	}

	xorAddr := &stun.XORMappedAddress{}
	mappedAddr := &stun.MappedAddress{}
	if err := xorAddr.GetFrom(&m); err == nil {
		return xorAddr.String(), nil
	}
	if err := mappedAddr.GetFrom(&m); err == nil {
		return mappedAddr.String(), nil
	}
	return "", fmt.Errorf("no address found")
}

type discoverConn struct {
	conn *net.UDPConn

	localAddr   net.Addr
	messageChan chan *Message
}

func listen() (*discoverConn, error) {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}

	return &discoverConn{
		conn:        conn,
		localAddr:   conn.LocalAddr(),
		messageChan: make(chan *Message, 10),
	}, nil
}

func (c *discoverConn) Close() error {
	if c.messageChan != nil {
		close(c.messageChan)
		c.messageChan = nil
	}
	return c.conn.Close()
}

func (c *discoverConn) readLoop() {
	for {
		buf := make([]byte, 1024)
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		buf = buf[:n]

		c.messageChan <- &Message{
			Body: buf,
			Addr: addr.String(),
		}
	}
}
