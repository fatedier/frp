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
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/fatedier/golib/net/stun"
)

var responseTimeout = 3 * time.Second

// If the localAddr is empty, it will listen on a random port.
func Discover(stunServers []string, localAddr string) ([]string, net.Addr, error) {
	discoverConn, err := listen(localAddr)
	if err != nil {
		return nil, nil, err
	}
	defer discoverConn.Close()

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
	conn      *net.UDPConn
	client    *stun.Client
	localAddr net.Addr
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
	client, err := stun.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &discoverConn{
		conn:      conn,
		client:    client,
		localAddr: conn.LocalAddr(),
	}, nil
}

func (c *discoverConn) Close() error {
	return c.conn.Close()
}

func (c *discoverConn) doSTUNRequest(addr string) (*stunResponse, error) {
	serverAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, err
	}
	transaction, err := stun.NewBindingTransaction(serverAddr)
	if err != nil {
		return nil, err
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(responseTimeout)); err != nil {
		return nil, err
	}
	response, err := c.client.Do(transaction)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("wait response from stun server timeout")
		}
		return nil, err
	}

	resp := &stunResponse{}
	if response.MappedAddr != nil {
		resp.externalAddr = response.MappedAddr.String()
	}
	if response.OtherAddr != nil {
		resp.otherAddr = response.OtherAddr.String()
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
