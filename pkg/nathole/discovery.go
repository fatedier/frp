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

	"github.com/pion/stun/v2"
)

var responseTimeout = 3 * time.Second

type Message struct {
	Body []byte
	Addr string
}

// If the localAddr is empty, it will listen on a random port.
func Discover(stunServers []string, localAddr string) ([]string, net.Addr, error) {
	// create a discoverConn and get response from messageChan
	discoverConn, err := listen(localAddr)
	if err != nil {
		return nil, nil, err
	}
	defer discoverConn.Close()

	go discoverConn.readLoop()

	addresses := make([]string, 0, len(stunServers))
	for _, addr := range stunServers {
		// get external address from stun server
		externalAddrs, err := discoverConn.discoverFromStunServer(addr)
		if err != nil {
			return nil, nil, err
		}
		addresses = append(addresses, externalAddrs...)
	}
	return addresses, discoverConn.localAddr, nil
}

type stunResponse struct {
	externalAddr string
	otherAddr    string
}

type discoverConn struct {
	conn *net.UDPConn

	localAddr   net.Addr
	messageChan chan *Message
}

func listen(localAddr string) (*discoverConn, error) {
	var local *net.UDPAddr
	if localAddr != "" {
		addr, err := net.ResolveUDPAddr("udp4", localAddr)
		if err != nil {
			return nil, err
		}
		local = addr
	}
	conn, err := net.ListenUDP("udp4", local)
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

func (c *discoverConn) doSTUNRequest(addr string) (*stunResponse, error) {
	serverAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	request, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		return nil, err
	}

	if err = request.NewTransactionID(); err != nil {
		return nil, err
	}
	if _, err := c.conn.WriteTo(request.Raw, serverAddr); err != nil {
		return nil, err
	}

	var m stun.Message
	select {
	case msg := <-c.messageChan:
		m.Raw = msg.Body
		if err := m.Decode(); err != nil {
			return nil, err
		}
	case <-time.After(responseTimeout):
		return nil, fmt.Errorf("wait response from stun server timeout")
	}
	xorAddrGetter := &stun.XORMappedAddress{}
	mappedAddrGetter := &stun.MappedAddress{}
	changedAddrGetter := ChangedAddress{}
	otherAddrGetter := &stun.OtherAddress{}

	resp := &stunResponse{}
	if err := mappedAddrGetter.GetFrom(&m); err == nil {
		resp.externalAddr = mappedAddrGetter.String()
	}
	if err := xorAddrGetter.GetFrom(&m); err == nil {
		resp.externalAddr = xorAddrGetter.String()
	}
	if err := changedAddrGetter.GetFrom(&m); err == nil {
		resp.otherAddr = changedAddrGetter.String()
	}
	if err := otherAddrGetter.GetFrom(&m); err == nil {
		resp.otherAddr = otherAddrGetter.String()
	}
	return resp, nil
}

func (c *discoverConn) discoverFromStunServer(addr string) ([]string, error) {
	resp, err := c.doSTUNRequest(addr)
	if err != nil {
		return nil, err
	}
	if resp.externalAddr == "" {
		return nil, fmt.Errorf("no external address found")
	}

	externalAddrs := make([]string, 0, 2)
	externalAddrs = append(externalAddrs, resp.externalAddr)

	if resp.otherAddr == "" {
		return externalAddrs, nil
	}

	// find external address from changed address
	resp, err = c.doSTUNRequest(resp.otherAddr)
	if err != nil {
		return nil, err
	}
	if resp.externalAddr != "" {
		externalAddrs = append(externalAddrs, resp.externalAddr)
	}
	return externalAddrs, nil
}
