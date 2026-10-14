// Copyright 2026 The frp Authors
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

package vnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// TestClientRouterDeleteRouteRequiresMatchingConnection verifies that a stale
// visitor cannot remove a replacement route registered under the same name.
func TestClientRouterDeleteRouteRequiresMatchingConnection(t *testing.T) {
	require := require.New(t)
	controller := NewController(v1.VirtualNetConfig{})

	_, route, err := net.ParseCIDR("10.1.0.1/32")
	require.NoError(err)

	oldConn, oldPeer := net.Pipe()
	t.Cleanup(func() {
		_ = oldConn.Close()
		_ = oldPeer.Close()
	})
	replacementConn, replacementPeer := net.Pipe()
	t.Cleanup(func() {
		_ = replacementConn.Close()
		_ = replacementPeer.Close()
	})

	controller.clientRouter.addRoute("vnet-visitor", []net.IPNet{*route}, oldConn)
	controller.clientRouter.addRoute("vnet-visitor", []net.IPNet{*route}, replacementConn)

	require.False(controller.UnregisterClientRoute("vnet-visitor", oldConn))
	got, err := controller.clientRouter.findConn(net.ParseIP("10.1.0.1"))
	require.NoError(err)
	require.Same(replacementConn, got)

	// The read loop for an old connection can exit after a replacement route
	// has already been registered. Its deferred cleanup must keep the new owner.
	controller.clientRouter.removeConnRoute(oldConn)
	got, err = controller.clientRouter.findConn(net.ParseIP("10.1.0.1"))
	require.NoError(err)
	require.Same(replacementConn, got)

	require.True(controller.UnregisterClientRoute("vnet-visitor", replacementConn))
	_, err = controller.clientRouter.findConn(net.ParseIP("10.1.0.1"))
	require.Error(err)
}
