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

package virtual

import (
	"context"
	"net"

	"github.com/fatedier/frp/client"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	netpkg "github.com/fatedier/frp/pkg/util/net"
)

type ClientOptions struct {
	Common           *v1.ClientCommonConfig
	Spec             *msg.ClientSpec
	HandleWorkConnCb func(*v1.ProxyBaseConfig, net.Conn, *msg.StartWorkConn) bool
}

type Client struct {
	l   *netpkg.InternalListener
	svr *client.Service
}

func NewClient(options ClientOptions) (*Client, error) {
	if options.Common != nil {
		options.Common.Complete()
	}

	ln := netpkg.NewInternalListener()

	serviceOptions := client.ServiceOptions{
		Common:     options.Common,
		ClientSpec: options.Spec,
		ConnectorCreator: func(context.Context, *v1.ClientCommonConfig) client.Connector {
			return &pipeConnector{
				peerListener: ln,
			}
		},
		HandleWorkConnCb: options.HandleWorkConnCb,
	}
	svr, err := client.NewService(serviceOptions)
	if err != nil {
		return nil, err
	}
	return &Client{
		l:   ln,
		svr: svr,
	}, nil
}

func (c *Client) PeerListener() net.Listener {
	return c.l
}

func (c *Client) UpdateProxyConfigurer(proxyCfgs []v1.ProxyConfigurer) {
	_ = c.svr.UpdateAllConfigurer(proxyCfgs, nil)
}

func (c *Client) Run(ctx context.Context) error {
	return c.svr.Run(ctx)
}

func (c *Client) Service() *client.Service {
	return c.svr
}

func (c *Client) Close() {
	c.svr.Close()
	c.l.Close()
}

type pipeConnector struct {
	peerListener *netpkg.InternalListener
}

func (pc *pipeConnector) Open() error {
	return nil
}

func (pc *pipeConnector) Connect() (net.Conn, error) {
	c1, c2 := net.Pipe()
	if err := pc.peerListener.PutConn(c1); err != nil {
		c1.Close()
		c2.Close()
		return nil, err
	}
	return c2, nil
}

func (pc *pipeConnector) Close() error {
	pc.peerListener.Close()
	return nil
}
