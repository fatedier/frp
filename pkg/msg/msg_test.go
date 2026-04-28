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

package msg

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV1MessageTypeIDsAreStable(t *testing.T) {
	require.Equal(t, byte('o'), TypeLogin)
	require.Equal(t, byte('1'), TypeLoginResp)
	require.Equal(t, byte('p'), TypeNewProxy)
	require.Equal(t, byte('2'), TypeNewProxyResp)
	require.Equal(t, byte('c'), TypeCloseProxy)
	require.Equal(t, byte('w'), TypeNewWorkConn)
	require.Equal(t, byte('r'), TypeReqWorkConn)
	require.Equal(t, byte('s'), TypeStartWorkConn)
	require.Equal(t, byte('v'), TypeNewVisitorConn)
	require.Equal(t, byte('3'), TypeNewVisitorConnResp)
	require.Equal(t, byte('h'), TypePing)
	require.Equal(t, byte('4'), TypePong)
	require.Equal(t, byte('u'), TypeUDPPacket)
	require.Equal(t, byte('i'), TypeNatHoleVisitor)
	require.Equal(t, byte('n'), TypeNatHoleClient)
	require.Equal(t, byte('m'), TypeNatHoleResp)
	require.Equal(t, byte('5'), TypeNatHoleSid)
	require.Equal(t, byte('6'), TypeNatHoleReport)
	require.Equal(t, byte('a'), TypeClientHelloAuto)
	require.Equal(t, byte('b'), TypeServerHelloAuto)
	require.Equal(t, byte('d'), TypeSelectTransport)
	require.Equal(t, byte('e'), TypeProbeTransport)
	require.Equal(t, byte('f'), TypeProbeTransportResp)
}

func TestMessageTypeMapIsCompleteAndUnique(t *testing.T) {
	require.Len(t, msgTypeMap, 23)

	msgTypes := make(map[reflect.Type]struct{}, len(msgTypeMap))

	for _, m := range msgTypeMap {
		msgType := reflect.TypeOf(m)
		require.NotContains(t, msgTypes, msgType)
		msgTypes[msgType] = struct{}{}
	}
}
