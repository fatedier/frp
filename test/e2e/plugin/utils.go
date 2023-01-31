package plugin

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	plugin "github.com/fatedier/frp/pkg/plugin/server"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
)

type Handler func(req *plugin.Request) *plugin.Response

type NewPluginRequest func() *plugin.Request

func NewHTTPPluginServer(port int, newFunc NewPluginRequest, handler Handler, tlsConfig *tls.Config) *httpserver.Server {
	return httpserver.New(
		httpserver.WithBindPort(port),
		httpserver.WithTLSConfig(tlsConfig),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			r := newFunc()
			buf, err := io.ReadAll(req.Body)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			log.Trace("plugin request: %s", string(buf))
			err = json.Unmarshal(buf, &r)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			resp := handler(r)
			buf, _ = json.Marshal(resp)
			log.Trace("plugin response: %s", string(buf))
			_, _ = w.Write(buf)
		})),
	)
}
