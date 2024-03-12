package features

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/onsi/ginkgo/v2"
	pp "github.com/pires/go-proxyproto"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/mock/server/httpserver"
	"github.com/fatedier/frp/test/e2e/mock/server/streamserver"
	"github.com/fatedier/frp/test/e2e/pkg/request"
	"github.com/fatedier/frp/test/e2e/pkg/rpc"
)

var _ = ginkgo.Describe("[Feature: Real IP]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("HTTP X-Forwarded-For", func() {
		vhostHTTPPort := f.AllocPort()
		serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhostHTTPPort = %d
		`, vhostHTTPPort)

		localPort := f.AllocPort()
		localServer := httpserver.New(
			httpserver.WithBindPort(localPort),
			httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte(req.Header.Get("X-Forwarded-For")))
			})),
		)
		f.RunServer("", localServer)

		clientConf := consts.DefaultClientConfig
		clientConf += fmt.Sprintf(`
		[[proxies]]
		name = "test"
		type = "http"
		localPort = %d
		customDomains = ["normal.example.com"]
		`, localPort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		framework.NewRequestExpect(f).Port(vhostHTTPPort).
			RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com")
			}).
			ExpectResp([]byte("127.0.0.1")).
			Ensure()
	})

	ginkgo.Describe("Proxy Protocol", func() {
		ginkgo.It("TCP", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			localPort := f.AllocPort()
			localServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(localPort),
				streamserver.WithCustomHandler(func(c net.Conn) {
					defer c.Close()
					rd := bufio.NewReader(c)
					ppHeader, err := pp.Read(rd)
					if err != nil {
						log.Errorf("read proxy protocol error: %v", err)
						return
					}

					for {
						if _, err := rpc.ReadBytes(rd); err != nil {
							return
						}

						buf := []byte(ppHeader.SourceAddr.String())
						_, _ = rpc.WriteBytes(c, buf)
					}
				}))
			f.RunServer("", localServer)

			remotePort := f.AllocPort()
			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "tcp"
			type = "tcp"
			localPort = %d
			remotePort = %d
			transport.proxyProtocolVersion = "v2"
			`, localPort, remotePort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(remotePort).Ensure(func(resp *request.Response) bool {
				log.Tracef("ProxyProtocol get SourceAddr: %s", string(resp.Content))
				addr, err := net.ResolveTCPAddr("tcp", string(resp.Content))
				if err != nil {
					return false
				}
				if addr.IP.String() != "127.0.0.1" {
					return false
				}
				return true
			})
		})

		ginkgo.It("HTTP", func() {
			vhostHTTPPort := f.AllocPort()
			serverConf := consts.DefaultServerConfig + fmt.Sprintf(`
		vhostHTTPPort = %d
		`, vhostHTTPPort)

			clientConf := consts.DefaultClientConfig

			localPort := f.AllocPort()
			var srcAddrRecord string
			localServer := streamserver.New(streamserver.TCP, streamserver.WithBindPort(localPort),
				streamserver.WithCustomHandler(func(c net.Conn) {
					defer c.Close()
					rd := bufio.NewReader(c)
					ppHeader, err := pp.Read(rd)
					if err != nil {
						log.Errorf("read proxy protocol error: %v", err)
						return
					}
					srcAddrRecord = ppHeader.SourceAddr.String()
				}))
			f.RunServer("", localServer)

			clientConf += fmt.Sprintf(`
			[[proxies]]
			name = "test"
			type = "http"
			localPort = %d
			customDomains = ["normal.example.com"]
			transport.proxyProtocolVersion = "v2"
			`, localPort)

			f.RunProcesses([]string{serverConf}, []string{clientConf})

			framework.NewRequestExpect(f).Port(vhostHTTPPort).RequestModify(func(r *request.Request) {
				r.HTTP().HTTPHost("normal.example.com")
			}).Ensure(framework.ExpectResponseCode(404))

			log.Tracef("ProxyProtocol get SourceAddr: %s", srcAddrRecord)
			addr, err := net.ResolveTCPAddr("tcp", srcAddrRecord)
			framework.ExpectNoError(err, srcAddrRecord)
			framework.ExpectEqualValues("127.0.0.1", addr.IP.String())
		})
	})
})
