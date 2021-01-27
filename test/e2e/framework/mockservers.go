package framework

import (
	"fmt"
	"os"

	"github.com/fatedier/frp/test/e2e/mock/echoserver"
	"github.com/fatedier/frp/test/e2e/pkg/port"
)

const (
	TCPEchoServerPort   = "TCPEchoServerPort"
	UDPEchoServerPort   = "UDPEchoServerPort"
	UDSEchoServerAddr   = "UDSEchoServerAddr"
	HTTPEchoServerPort  = "HTTPEchoServerPort"
	HTTPSEchoServerPort = "HTTPSEchoServerPort"
	WSEchoServerPort    = "WSEchoServerPort"
	WSSEchoServerPort   = "WSSEchoServerPort"
)

type MockServers struct {
	tcpEchoServer   *echoserver.Server
	udpEchoServer   *echoserver.Server
	udsEchoServer   *echoserver.Server
	httpEchoServer  *echoserver.Server
	httpsEchoServer *echoserver.Server
	wsEchoServer    *echoserver.Server
	wssEchoServer   *echoserver.Server
}

func NewMockServers(portAllocator *port.Allocator) *MockServers {
	s := &MockServers{}
	tcpPort := portAllocator.Get()
	udpPort := portAllocator.Get()
	httpPort := portAllocator.Get()
	httpsPort := portAllocator.Get()
	wsPort := portAllocator.Get()
	wssPort := portAllocator.Get()

	s.tcpEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.TCP,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(tcpPort),
		RepeatNum: 1,
	})
	s.udpEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.UDP,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(udpPort),
		RepeatNum: 1,
	})

	udsIndex := portAllocator.Get()
	udsAddr := fmt.Sprintf("%s/frp_echo_server_%d.sock", os.TempDir(), udsIndex)
	os.Remove(udsAddr)
	s.udsEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.Unix,
		BindAddr:  udsAddr,
		RepeatNum: 1,
	})

	s.httpEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.HTTP,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(httpPort),
		RepeatNum: 1,
	})

	s.httpsEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.HTTPS,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(httpsPort),
		RepeatNum: 1,
	})

	s.wsEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.WS,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(wsPort),
		RepeatNum: 1,
	})
	s.wssEchoServer = echoserver.New(echoserver.Options{
		Type:      echoserver.WSS,
		BindAddr:  "127.0.0.1",
		BindPort:  int32(wssPort),
		RepeatNum: 1,
	})

	return s
}

func (m *MockServers) Run() error {
	if err := m.tcpEchoServer.Run(); err != nil {
		return err
	}
	if err := m.udpEchoServer.Run(); err != nil {
		return err
	}
	if err := m.udsEchoServer.Run(); err != nil {
		return err
	}
	if err := m.httpEchoServer.Run(); err != nil {
		return err
	}
	if err := m.httpsEchoServer.Run(); err != nil {
		return err
	}
	if err := m.wsEchoServer.Run(); err != nil {
		return err
	}
	if err := m.wssEchoServer.Run(); err != nil {
		return err
	}
	return nil
}

func (m *MockServers) Close() {
	m.tcpEchoServer.Close()
	m.udpEchoServer.Close()
	m.udsEchoServer.Close()
	m.httpEchoServer.Close()
	m.httpsEchoServer.Close()
	m.wsEchoServer.Close()
	m.wssEchoServer.Close()
	os.Remove(m.udsEchoServer.GetOptions().BindAddr)
}

func (m *MockServers) GetTemplateParams() map[string]interface{} {
	ret := make(map[string]interface{})
	ret[TCPEchoServerPort] = m.tcpEchoServer.GetOptions().BindPort
	ret[UDPEchoServerPort] = m.udpEchoServer.GetOptions().BindPort
	ret[UDSEchoServerAddr] = m.udsEchoServer.GetOptions().BindAddr
	ret[HTTPEchoServerPort] = m.httpEchoServer.GetOptions().BindPort
	ret[HTTPSEchoServerPort] = m.httpsEchoServer.GetOptions().BindPort
	ret[WSEchoServerPort] = m.wsEchoServer.GetOptions().BindPort
	ret[WSSEchoServerPort] = m.wssEchoServer.GetOptions().BindPort
	return ret
}

func (m *MockServers) GetParam(key string) interface{} {
	params := m.GetTemplateParams()
	if v, ok := params[key]; ok {
		return v
	}
	return nil
}
