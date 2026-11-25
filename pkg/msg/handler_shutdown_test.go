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
	"bytes"
	"errors"
	"io"
	"testing"
)

// Preserve the original exported constructor function type.
var _ func(ReadWriter) *Dispatcher = NewDispatcher

func TestDispatcherEnqueueRejectsStoppedDispatcher(t *testing.T) {
	c := newRegressionControl(t)
	d := NewDispatcher(c)
	d.stop()

	// Model shutdown after Send's initial check: the stop and buffered send
	// cases are both ready. A send selected here must still return EOF.
	if err := d.enqueue(&ReqWorkConn{}); !errors.Is(err, io.EOF) {
		t.Fatalf("enqueue racing with shutdown: %v, want EOF", err)
	}
	// This single call checks that a stopped dispatcher's enqueue returns EOF; it
	// does not guarantee exercising the send case and post-enqueue check on every run.
	if len(d.sendCh) > 0 {
		t.Log("selected the send case; post-enqueue check returned EOF")
	}
}

func TestDispatcherAcceptsReadWriterWithoutClose(t *testing.T) {
	t.Run("codec factory", func(t *testing.T) {
		d := NewDispatcher(NewV1ReadWriter(&bytes.Buffer{}))
		d.Run()
		regressionAwait(t, d.Done(), "non-closable codec EOF")
	})

	t.Run("write failure", func(t *testing.T) {
		c := newRegressionControl(t)
		regressionWarm(t, c)
		// Hide Close to model an existing caller implementing only ReadWriter.
		d := NewDispatcher(struct{ ReadWriter }{c})
		d.Run()
		t.Cleanup(func() {
			c.transport.close()
			regressionAwait(t, d.Done(), "externally interrupted reader")
		})

		c.transport.arm()
		if err := d.Send(&ReqWorkConn{}); err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
		if err := regressionReceiveWrite(t, c); !errors.Is(err, errRegressionWriteTimeout) {
			t.Fatalf("injected write failure: %v", err)
		}
		regressionAwait(t, d.stopCh, "write failure stops sends")
		if err := d.Send(&ReqWorkConn{}); !errors.Is(err, io.EOF) {
			t.Fatalf("Send after write failure: %v, want EOF", err)
		}
		if c.closeCalls.Load() != 0 {
			t.Fatal("dispatcher closed a connection hidden by ReadWriter")
		}
		select {
		case <-d.Done():
			t.Fatal("Done closed before the non-closable reader returned")
		default:
		}
	})
}
