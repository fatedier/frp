package basic

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	"github.com/fatedier/frp/test/e2e/pkg/utils"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: HTTP]", func() {
	f := framework.NewDefaultFramework()

	getDefaultServerConf := func(vhostHTTPPort int) string {
		conf := consts.DefaultServerConfig + `
		vhost_http_port = %d
		`
		return fmt.Sprintf(conf, vhostHTTPPort)
	}
	newHTTPServer := func(port int, respContent string) *httpserver.Server {
		return httpserver.New(
			httpserver.WithBindPort(port),
			httpserver.WithHandler(framework.SpecifiedHTTPBodyHandler([]byte(respContent))),
		)
	}

	It("HTTP route by locations", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		fooPort := f.AllocPort()
		f.RunServer("", newHTTPServer(fooPort, "foo"))

		barPort := f.AllocPort()
		f.RunServer("", newHTTPServer(barPort, "bar"))

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[foo]
			type = http
			local_port = %d
			custom_domains = normal.example.com
			locations = /,/foo

			[bar]
			type = http
			local_port = %d
			custom_domains = normal.example.com
			locations = /bar
			`, fooPort, barPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// foo path
		framework.NewRequestExpect(f).Explain("foo path").Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com").HTTPPath("/foo")
			}).
			ExpectResp([]byte("foo")).
			Ensure()

		// bar path
		framework.NewRequestExpect(f).Explain("bar path").Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com").HTTPPath("/bar")
			}).
			ExpectResp([]byte("bar")).
			Ensure()

		// other path
		framework.NewRequestExpect(f).Explain("other path").Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com").HTTPPath("/other")
			}).
			ExpectResp([]byte("foo")).
			Ensure()
	})

	It("HTTP Basic Auth", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[test]
			type = http
			local_port = {{ .%s }}
			custom_domains = normal.example.com
			http_user = test
			http_pwd = test
			`, framework.HTTPSimpleServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// not set auth header
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com")
			}).
			Ensure(framework.ExpectResponseCode(401))

		// set incorrect auth header
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com").HTTPHeaders(map[string]string{
					"Authorization": utils.BasicAuth("test", "invalid"),
				})
			}).
			Ensure(framework.ExpectResponseCode(401))

		// set correct auth header
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com").HTTPHeaders(map[string]string{
					"Authorization": utils.BasicAuth("test", "test"),
				})
			}).
			Ensure()
	})

	It("Wildcard domain", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[test]
			type = http
			local_port = {{ .%s }}
			custom_domains = *.example.com
			`, framework.HTTPSimpleServerPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// not match host
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("not-match.test.com")
			}).
			Ensure(framework.ExpectResponseCode(404))

		// test.example.com match *.example.com
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("test.example.com")
			}).
			Ensure()

		// sub.test.example.com match *.example.com
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("sub.test.example.com")
			}).
			Ensure()
	})

	It("Subdomain", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)
		serverConf += `
		subdomain_host = example.com
		`

		fooPort := f.AllocPort()
		f.RunServer("", newHTTPServer(fooPort, "foo"))

		barPort := f.AllocPort()
		f.RunServer("", newHTTPServer(barPort, "bar"))

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[foo]
			type = http
			local_port = %d
			subdomain = foo

			[bar]
			type = http
			local_port = %d
			subdomain = bar
			`, fooPort, barPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// foo
		framework.NewRequestExpect(f).Explain("foo subdomain").Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("foo.example.com")
			}).
			ExpectResp([]byte("foo")).
			Ensure()

		// bar
		framework.NewRequestExpect(f).Explain("bar subdomain").Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("bar.example.com")
			}).
			ExpectResp([]byte("bar")).
			Ensure()
	})

	It("Modify headers", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		localPort := f.AllocPort()
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte(req.Header.Get("X-From-Where")))
			})),
		)
		f.RunServer("", localServer)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[test]
			type = http
			local_port = %d
			custom_domains = normal.example.com
			header_X-From-Where = frp
			`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// not set auth header
		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com")
			}).
			ExpectResp([]byte("frp")). // local http server will write this X-From-Where header to response body
			Ensure()
	})

	It("Host Header Rewrite", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		localPort := f.AllocPort()
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte(req.Host))
			})),
		)
		f.RunServer("", localServer)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[test]
			type = http
			local_port = %d
			custom_domains = normal.example.com
			host_header_rewrite = rewrite.example.com
			`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com")
			}).
			ExpectResp([]byte("rewrite.example.com")). // local http server will write host header to response body
			Ensure()
	})

	It("Websocket protocol", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := getDefaultServerConf(vhostHTTPPort)

		upgrader := websocket.Upgrader{}

		localPort := f.AllocPort()
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				c, err := upgrader.Upgrade(w, req, nil)
				if err != nil {
					return
				}
				defer c.Close()
				for {
					mt, message, err := c.ReadMessage()
					if err != nil {
						break
					}
					err = c.WriteMessage(mt, message)
					if err != nil {
						break
					}
				}
			})),
		)

		f.RunServer("", localServer)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
			[test]
			type = http
			local_port = %d
			custom_domains = 127.0.0.1
			`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		u := url.URL{Scheme: "ws", Host: "127.0.0.1:" + strconv.Itoa(vhostHTTPPort)}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		framework.ExpectNoError(err)

		err = c.WriteMessage(websocket.TextMessage, []byte(consts.TestString))
		framework.ExpectNoError(err)

		_, msg, err := c.ReadMessage()
		framework.ExpectNoError(err)
		framework.ExpectEqualValues(consts.TestString, string(msg))
	})
})
