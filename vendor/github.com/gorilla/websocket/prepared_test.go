// Copyright 2017 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bytes"
	"compress/flate"
	"math/rand"
	"testing"
)

var preparedMessageTests = []struct {
	messageType            int
	isServer               bool
	enableWriteCompression bool
	compressionLevel       int
}{
	// Server
	{TextMessage, true, false, flate.BestSpeed},
	{TextMessage, true, true, flate.BestSpeed},
	{TextMessage, true, true, flate.BestCompression},
	{PingMessage, true, false, flate.BestSpeed},
	{PingMessage, true, true, flate.BestSpeed},

	// Client
	{TextMessage, false, false, flate.BestSpeed},
	{TextMessage, false, true, flate.BestSpeed},
	{TextMessage, false, true, flate.BestCompression},
	{PingMessage, false, false, flate.BestSpeed},
	{PingMessage, false, true, flate.BestSpeed},
}

func TestPreparedMessage(t *testing.T) {
	for _, tt := range preparedMessageTests {
		var data = []byte("this is a test")
		var buf bytes.Buffer
		c := newConn(fakeNetConn{Reader: nil, Writer: &buf}, tt.isServer, 1024, 1024)
		if tt.enableWriteCompression {
			c.newCompressionWriter = compressNoContextTakeover
		}
		c.SetCompressionLevel(tt.compressionLevel)

		// Seed random number generator for consistent frame mask.
		rand.Seed(1234)

		if err := c.WriteMessage(tt.messageType, data); err != nil {
			t.Fatal(err)
		}
		want := buf.String()

		pm, err := NewPreparedMessage(tt.messageType, data)
		if err != nil {
			t.Fatal(err)
		}

		// Scribble on data to ensure that NewPreparedMessage takes a snapshot.
		copy(data, "hello world")

		// Seed random number generator for consistent frame mask.
		rand.Seed(1234)

		buf.Reset()
		if err := c.WritePreparedMessage(pm); err != nil {
			t.Fatal(err)
		}
		got := buf.String()

		if got != want {
			t.Errorf("write message != prepared message for %+v", tt)
		}
	}
}
