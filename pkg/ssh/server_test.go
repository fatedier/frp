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

package ssh

import (
	"encoding/binary"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	cryptossh "golang.org/x/crypto/ssh"
)

func TestParseExecPayload(t *testing.T) {
	payload := cryptossh.Marshal(&execPayload{Command: "tcp --remote_port 6000"})

	got, ok := parseExecPayload(payload)

	require.True(t, ok)
	require.Equal(t, "tcp --remote_port 6000", got)
}

func TestParseExecPayloadRejectsMalformedPayloads(t *testing.T) {
	overflowLength := make([]byte, 5)
	binary.BigEndian.PutUint32(overflowLength[:4], ^uint32(0))

	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{
			name:    "empty",
			payload: nil,
		},
		{
			name:    "short length prefix",
			payload: []byte{0, 0, 0},
		},
		{
			name:    "declared length exceeds remaining payload",
			payload: []byte{0, 0, 0, 2, 'x'},
		},
		{
			name:    "overflow length",
			payload: overflowLength,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				got string
				ok  bool
			)
			require.NotPanics(t, func() {
				got, ok = parseExecPayload(tc.payload)
			})
			require.False(t, ok)
			require.Empty(t, got)
		})
	}
}

type trackingChannel struct {
	active     atomic.Int32
	concurrent atomic.Bool
}

func (c *trackingChannel) Read([]byte) (int, error) { return 0, io.EOF }

func (c *trackingChannel) Write(p []byte) (int, error) {
	if c.active.Add(1) != 1 {
		c.concurrent.Store(true)
	}
	time.Sleep(time.Millisecond)
	c.active.Add(-1)
	return len(p), nil
}

func (c *trackingChannel) Close() error                                   { return nil }
func (c *trackingChannel) CloseWrite() error                              { return nil }
func (c *trackingChannel) SendRequest(string, bool, []byte) (bool, error) { return false, nil }
func (c *trackingChannel) Stderr() io.ReadWriter                          { return nil }

func TestWriteToClientSerializesChannelWrites(t *testing.T) {
	channel := &trackingChannel{}
	s := &TunnelServer{firstChannel: channel}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			<-start
			s.writeToClient("message")
		})
	}
	close(start)
	wg.Wait()

	if channel.concurrent.Load() {
		t.Fatal("channel writes were concurrent")
	}
}
