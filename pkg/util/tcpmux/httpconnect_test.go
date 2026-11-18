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

package tcpmux

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	netpkg "github.com/fatedier/frp/pkg/util/net"
	"github.com/fatedier/frp/pkg/util/vhost"
)

func TestHTTPConnectTCPMuxerPipelinedData(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(fmt.Sprintf("passthrough=%v", passthrough), func(t *testing.T) {
			listener := netpkg.NewInternalListener()
			mux, err := NewHTTPConnectTCPMuxer(listener, passthrough, 5*time.Second)
			require.NoError(t, err)
			defer mux.Close()

			l, err := mux.Listen(context.Background(), &vhost.RouteConfig{Domain: "target.example.com"})
			require.NoError(t, err)

			client, incoming := net.Pipe()
			defer client.Close()
			require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, listener.PutConn(incoming))

			request := "CONNECT target.example.com:443 HTTP/1.1\r\nHost: target.example.com:443\r\n\r\n"
			writeErr := make(chan error, 1)
			go func() {
				// One pipe write puts the tunnel data in the request reader's buffer.
				_, err := io.WriteString(client, request+"early data")
				writeErr <- err
			}()

			clientReader := bufio.NewReader(client)
			if !passthrough {
				resp, err := http.ReadResponse(clientReader, nil)
				require.NoError(t, err)
				resp.Body.Close()
				require.Equal(t, http.StatusOK, resp.StatusCode)
			}

			accepted, err := l.Accept()
			require.NoError(t, err)
			defer accepted.Close()
			require.NoError(t, accepted.SetDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, <-writeErr)

			acceptedReader := bufio.NewReader(accepted)
			if passthrough {
				req, err := http.ReadRequest(acceptedReader)
				require.NoError(t, err)
				require.Equal(t, http.MethodConnect, req.Method)
			}
			data := make([]byte, len("early data"))
			_, err = io.ReadFull(acceptedReader, data)
			require.NoError(t, err)
			require.Equal(t, "early data", string(data))
		})
	}
}
